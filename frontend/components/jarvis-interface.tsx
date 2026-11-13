"use client";

import Link from "next/link";
import { useState } from "react";

import { ChatPanel } from "@/components/chat-panel";
import { HealthPanel } from "@/components/health-panel";
import { ToolsPanel } from "@/components/tools-panel";
import { VoiceAssistant } from "@/components/voice-assistant";

type Panel = "text" | "system" | null;

export function JarvisInterface() {
  const [activePanel, setActivePanel] = useState<Panel>(null);

  return (
    <main className="jarvis-hud">
 <Link href="/lab" style={{position:"fixed",bottom:16,left:16,zIndex:30,color:"#67e8f9"}}>Laboratório de voz</Link>
      <VoiceAssistant
        onOpenText={() => setActivePanel("text")}
        onOpenSystem={() => setActivePanel("system")}
      />

      {activePanel ? (
        <div className="hud-overlay" role="dialog" aria-modal="true">
          <button
            type="button"
            className="hud-scrim"
            aria-label="Fechar painel"
            onClick={() => setActivePanel(null)}
          />
          <aside className="hud-panel">
            <div className="hud-panel-header">
              <span>{activePanel === "text" ? "Modo Texto" : "Sistema"}</span>
              <button
                type="button"
                className="hud-close"
                onClick={() => setActivePanel(null)}
              >
                Fechar
              </button>
            </div>

            <div className="hud-panel-body">
              {activePanel === "text" ? (
                <ChatPanel />
              ) : (
                <>
                  <ToolsPanel />
                  <HealthPanel />
                </>
              )}
            </div>
          </aside>
        </div>
      ) : null}
    </main>
  );
}
