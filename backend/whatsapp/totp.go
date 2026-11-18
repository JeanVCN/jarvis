package main

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"database/sql"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow/types"
)

const (
	totpPeriodSeconds = int64(30)
	totpAuthTTL       = 10 * time.Minute
	totpFailureWindow = 15 * time.Minute
	totpLockout       = 15 * time.Minute
	totpMaxFailures   = 5
)

var totpCodePattern = regexp.MustCompile(`^[0-9]{6}$`)

type contextRowQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func createTOTPTable(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS jarvis_totp (
		id INTEGER PRIMARY KEY CHECK (id = 1),
		secret TEXT NOT NULL,
		last_step INTEGER NOT NULL DEFAULT -1,
		authorized_until INTEGER NOT NULL DEFAULT 0,
		failed_attempts INTEGER NOT NULL DEFAULT 0,
		failure_window_started INTEGER NOT NULL DEFAULT 0,
		locked_until INTEGER NOT NULL DEFAULT 0,
		challenge_pending INTEGER NOT NULL DEFAULT 0
	)`)
	if err != nil {
		return err
	}
	// Migrate databases created by the earlier TOTP draft.
	_, err = db.ExecContext(ctx, "ALTER TABLE jarvis_totp ADD COLUMN challenge_pending INTEGER NOT NULL DEFAULT 0")
	if err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
		return err
	}
	return nil
}

func initializeTOTP(ctx context.Context, db *sql.DB) error {
	if err := createTOTPTable(ctx, db); err != nil {
		return err
	}
	// A process restart always closes the temporary authorization window.
	_, err := db.ExecContext(ctx, "UPDATE jarvis_totp SET authorized_until = 0 WHERE id = 1")
	return err
}

func totpAuthorization(ctx context.Context, queryer contextRowQuerier) (configured, authorized bool, err error) {
	var authorizedUntil int64
	err = queryer.QueryRowContext(ctx, "SELECT authorized_until FROM jarvis_totp WHERE id = 1").Scan(&authorizedUntil)
	if errors.Is(err, sql.ErrNoRows) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	return true, time.Now().Unix() < authorizedUntil, nil
}

func isTOTPCommand(text string) bool {
	fields := strings.Fields(strings.TrimSpace(text))
	return len(fields) > 0 && strings.EqualFold(fields[0], "/2fa")
}

func (r messageRouter) handleTOTPCommand(typed bool) string {
	if !typed {
		return "A autenticação deve ser digitada: envie `/code` e depois responda com os seis dígitos. Não encaminhei esse conteúdo."
	}
	return "Para autenticar, envie `/code` e depois responda somente com os seis dígitos do aplicativo autenticador."
}

func (r messageRouter) verifyTOTP(ctx context.Context, code string, now time.Time) (string, bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", false, err
	}
	defer tx.Rollback()

	var secret string
	var lastStep, failures, windowStarted, lockedUntil int64
	err = tx.QueryRowContext(ctx, `SELECT secret,last_step,failed_attempts,failure_window_started,locked_until
		FROM jarvis_totp WHERE id = 1`).Scan(&secret, &lastStep, &failures, &windowStarted, &lockedUntil)
	if errors.Is(err, sql.ErrNoRows) {
		return "O autenticador ainda não foi configurado. Pare o serviço e execute `JARVIS_WHATSAPP_OWNER=... go run . totp-setup` no terminal local.", false, nil
	}
	if err != nil {
		return "", false, err
	}
	nowUnix := now.Unix()
	if nowUnix < lockedUntil {
		_ = tx.Commit()
		return "Muitas tentativas inválidas. A autenticação ficará bloqueada por alguns minutos.", false, nil
	}
	if windowStarted == 0 || nowUnix-windowStarted >= int64(totpFailureWindow/time.Second) {
		failures = 0
		windowStarted = nowUnix
	}
	step, valid := matchingTOTPStep(secret, code, now, lastStep)
	if !valid {
		failures++
		lockedUntil = 0
		if failures >= totpMaxFailures {
			failures = 0
			windowStarted = nowUnix
			lockedUntil = nowUnix + int64(totpLockout/time.Second)
		}
		if _, err := tx.ExecContext(ctx, `UPDATE jarvis_totp SET failed_attempts=?,failure_window_started=?,locked_until=? WHERE id=1`, failures, windowStarted, lockedUntil); err != nil {
			return "", false, err
		}
		if err := tx.Commit(); err != nil {
			return "", false, err
		}
		if lockedUntil > nowUnix {
			return "Muitas tentativas inválidas. A autenticação ficará bloqueada por alguns minutos.", false, nil
		}
		return "Código inválido ou já utilizado. Confira o relógio do autenticador e tente novamente.", false, nil
	}

	result, err := tx.ExecContext(ctx, `UPDATE jarvis_totp SET last_step=?,authorized_until=?,failed_attempts=0,
		failure_window_started=0,locked_until=0 WHERE id=1 AND last_step=?`, step, nowUnix+int64(totpAuthTTL/time.Second), lastStep)
	if err != nil {
		return "", false, err
	}
	changed, err := result.RowsAffected()
	if err != nil {
		return "", false, err
	}
	if changed != 1 {
		return "Código já utilizado. Gere o próximo código no autenticador.", false, nil
	}
	if err := tx.Commit(); err != nil {
		return "", false, err
	}
	return "Autenticação concluída.", true, nil
}

func matchingTOTPStep(encodedSecret, code string, now time.Time, lastStep int64) (int64, bool) {
	if !totpCodePattern.MatchString(code) {
		return 0, false
	}
	secret, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(encodedSecret)
	if err != nil || len(secret) == 0 {
		return 0, false
	}
	currentStep := now.Unix() / totpPeriodSeconds
	matchedStep := int64(-1)
	for step := currentStep - 1; step <= currentStep+1; step++ {
		expected := totpValue(secret, step)
		if subtle.ConstantTimeCompare([]byte(expected), []byte(code)) == 1 && step > lastStep && step > matchedStep {
			matchedStep = step
		}
	}
	if matchedStep < 0 {
		return 0, false
	}
	return matchedStep, true
}

func totpValue(secret []byte, step int64) string {
	var counter [8]byte
	binary.BigEndian.PutUint64(counter[:], uint64(step))
	mac := hmac.New(sha1.New, secret)
	_, _ = mac.Write(counter[:])
	digest := mac.Sum(nil)
	offset := digest[len(digest)-1] & 0x0f
	value := binary.BigEndian.Uint32(digest[offset:offset+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", value%1_000_000)
}

func manageTOTP(ctx context.Context, command string) error {
	number := strings.TrimPrefix(strings.TrimSpace(os.Getenv("JARVIS_WHATSAPP_OWNER")), "+")
	if !phoneNumber.MatchString(number) {
		return errors.New("defina JARVIS_WHATSAPP_OWNER com o número autorizado, incluindo país e DDD")
	}
	owner := types.NewJID(number, types.DefaultUserServer)
	dir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	dir = filepath.Join(dir, "jarvis", "whatsapp")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	if err := os.Chmod(dir, 0700); err != nil {
		return err
	}
	dbPath, err := accountStorePath(ctx, dir, owner)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(dbPath, os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	if err := file.Chmod(0600); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	dsn := url.URL{Scheme: "file", Path: dbPath, RawQuery: "_foreign_keys=on&_busy_timeout=5000"}
	db, err := sql.Open("sqlite3", dsn.String())
	if err != nil {
		return err
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if err := createTOTPTable(ctx, db); err != nil {
		return err
	}

	var secretBytes [20]byte
	if _, err := rand.Read(secretBytes[:]); err != nil {
		return err
	}
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secretBytes[:])
	if command == "totp-setup" {
		_, err = db.ExecContext(ctx, `INSERT INTO jarvis_totp(id,secret) VALUES(1,?)`, secret)
	} else {
		_, err = db.ExecContext(ctx, `INSERT INTO jarvis_totp(id,secret) VALUES(1,?)
			ON CONFLICT(id) DO UPDATE SET secret=excluded.secret,last_step=-1,authorized_until=0,
			failed_attempts=0,failure_window_started=0,locked_until=0`, secret)
	}
	if err != nil {
		if command == "totp-setup" {
			return errors.New("o autenticador já está configurado; pare o serviço e use totp-reset para trocar a chave")
		}
		return err
	}

	label := url.PathEscape("Jarvis:" + number)
	query := url.Values{
		"secret":    {secret},
		"issuer":    {"Jarvis"},
		"algorithm": {"SHA1"},
		"digits":    {"6"},
		"period":    {"30"},
	}
	uri := "otpauth://totp/" + label + "?" + query.Encode()
	fmt.Fprintln(os.Stdout, "Adicione esta conta ao Google Authenticator agora. Não compartilhe nem registre o segredo:")
	qrterminal.GenerateHalfBlock(uri, qrterminal.L, os.Stdout)
	fmt.Fprintf(os.Stdout, "\nConta: Jarvis (%s)\nChave manual: %s\n", number, secret)
	fmt.Fprintln(os.Stdout, "Depois de escanear, encerre este comando. O segredo fica no SQLite local com permissões restritas.")
	return nil
}
