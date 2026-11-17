#!/usr/bin/env python3
"""Build and install Jarvis user service units without enabling them."""
import argparse
import os
from pathlib import Path
import re
import secrets
import shutil
import subprocess
import sys


ROOT = Path(__file__).resolve().parents[1]
OWNER_PATTERN = re.compile(r"^[1-9][0-9]{7,14}$")


def fail(message: str) -> None:
    raise SystemExit(f"Erro: {message}")


def run(command: list[str], cwd: Path | None = None) -> None:
    result = subprocess.run(command, cwd=cwd, check=False)
    if result.returncode:
        fail(f"comando falhou ({result.returncode}): {command[0]}")


def systemd_quote(value: str) -> str:
    return '"' + value.replace("\\", "\\\\").replace('"', '\\"') + '"'


def main() -> None:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--owner",
        action="append",
        default=[],
        metavar="NUMERO",
        help="número WhatsApp com país e DDD; pode ser repetido para várias instâncias",
    )
    parser.add_argument(
        "--with-opencode-bridge",
        action="store_true",
        help="instala também o unit do bridge (requer opencode-bridge já compilado)",
    )
    parser.add_argument(
        "--with-web",
        action="store_true",
        help="instala a unidade para Core, PostgreSQL e frontend sem o agente de voz contínua",
    )
    args = parser.parse_args()

    if sys.platform != "linux":
        fail("os serviços systemd estão disponíveis somente no Linux")
    systemctl = shutil.which("systemctl")
    if not systemctl:
        fail("systemctl não foi encontrado")
    if not args.owner:
        fail("informe ao menos um --owner com número WhatsApp")
    for owner in args.owner:
        if not OWNER_PATTERN.fullmatch(owner):
            fail(f"número inválido: {owner}; use de 8 a 15 dígitos com código do país")

    go = shutil.which("go")
    if not go:
        fail("Go não foi encontrado no PATH; instale Go e tente novamente")
    node = None
    if args.with_opencode_bridge:
        node = shutil.which("node")
        if not node:
            fail("Node.js não foi encontrado no PATH; instale as dependências e tente novamente")
        if not (ROOT / "opencode-bridge" / "dist" / "server.js").is_file():
            fail("bridge sem build; execute npm ci e npm run build em opencode-bridge")
    docker = None
    npm = None
    if args.with_web:
        docker = shutil.which("docker")
        npm = shutil.which("npm")
        if not docker or not npm:
            fail("o perfil web requer Docker Compose e npm no PATH")
        compose = subprocess.run([docker, "compose", "version"], check=False, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        if compose.returncode:
            fail("Docker Compose não está disponível para o usuário atual")
        if not (ROOT / "frontend" / "node_modules" / "next").is_dir():
            fail("frontend sem dependências; execute npm ci e npm run build em frontend")

    config_home = Path.home() / ".config"
    user_units = config_home / "systemd" / "user"
    jarvis_config = config_home / "jarvis"
    binaries = Path.home() / ".local" / "lib" / "jarvis" / "whatsapp"
    user_units.mkdir(parents=True, exist_ok=True, mode=0o700)
    jarvis_config.mkdir(parents=True, exist_ok=True, mode=0o700)
    binaries.mkdir(parents=True, exist_ok=True, mode=0o700)
    os.chmod(jarvis_config, 0o700)
    os.chmod(binaries, 0o700)

    module = ROOT / "backend" / "whatsapp"
    for owner in dict.fromkeys(args.owner):
        target = binaries / owner
        temporary = binaries / (owner + ".new")
        temporary.unlink(missing_ok=True)
        run([go, "build", "-trimpath", "-o", str(temporary), "."], cwd=module)
        temporary.chmod(0o755)
        os.replace(temporary, target)
        env_file = jarvis_config / f"whatsapp-{owner}.env"
        if not env_file.exists():
            env_file.write_text(f"JARVIS_WHATSAPP_OWNER={owner}\n")
        env_file.chmod(0o600)

    whatsapp_unit = ROOT / "deploy" / "systemd" / "jarvis-whatsapp@.service"
    shutil.copyfile(whatsapp_unit, user_units / whatsapp_unit.name)
    (user_units / whatsapp_unit.name).chmod(0o644)

    if args.with_opencode_bridge:
        bridge_env = jarvis_config / "opencode-bridge.env"
        if not bridge_env.exists():
            bridge_env.write_text(
                "JARVIS_OPENCODE_BRIDGE_TOKEN=" + secrets.token_hex(32) + "\n"
                "JARVIS_OPENCODE_PROJECTS=\n"
                "JARVIS_OPENCODE_WRITE_ENABLED=false\n"
                "JARVIS_OPENCODE_BRIDGE_PORT=18110\n"
            )
        bridge_env.chmod(0o600)
        bridge_template = (ROOT / "deploy" / "systemd" / "jarvis-opencode-bridge.service.in").read_text()
        bridge_unit = bridge_template.replace("@JARVIS_BRIDGE_DIR@", str(ROOT / "opencode-bridge")).replace(
            "@NODE_BINARY@", systemd_quote(node)
        )
        target = user_units / "jarvis-opencode-bridge.service"
        target.write_text(bridge_unit)
        target.chmod(0o644)

    if args.with_web:
        web_template = (ROOT / "deploy" / "systemd" / "jarvis-web.service.in").read_text()
        service_paths = [str(Path(binary).resolve().parent) for binary in (go, node, npm, docker, sys.executable) if binary]
        service_paths.extend(["/usr/local/sbin", "/usr/local/bin", "/usr/sbin", "/usr/bin", "/sbin", "/bin"])
        service_path = os.pathsep.join(dict.fromkeys(service_paths))
        web_unit = web_template.replace("@JARVIS_ROOT@", str(ROOT)).replace(
            "@JARVIS_SCRIPT@", systemd_quote(str(ROOT / "scripts" / "jarvis"))
        ).replace("@SERVICE_PATH@", systemd_quote("PATH=" + service_path))
        target = user_units / "jarvis-web.service"
        target.write_text(web_unit)
        target.chmod(0o644)

    run([systemctl, "--user", "daemon-reload"])
    print("Units instalados e recarregados; nenhum serviço foi iniciado ou habilitado.")
    for owner in dict.fromkeys(args.owner):
        print(f"Edite {jarvis_config / f'whatsapp-{owner}.env'} e habilite com:")
        print(f"  systemctl --user enable --now jarvis-whatsapp@{owner}.service")
    if args.with_opencode_bridge:
        print(f"Configure {jarvis_config / 'opencode-bridge.env'} com a allowlist JSON de projetos antes de iniciar o bridge.")
        print("  systemctl --user enable --now jarvis-opencode-bridge.service")
    if args.with_web:
        print("O perfil web inicia Core/PostgreSQL/frontend sem captura contínua de voz.")
        if args.with_opencode_bridge:
            print("Ele usa o bridge separado; não inicie outra instância do bridge pelo scripts/jarvis run.")
        print("  systemctl --user enable --now jarvis-web.service")
    print("Para iniciar estes serviços no boot sem entrar na sessão gráfica, habilite linger para seu usuário.")


if __name__ == "__main__":
    main()
