import type { Metadata } from "next";
import { headers } from "next/headers";
import { AuthControls } from "@/components/auth-controls";
import "./globals.css";

export const metadata: Metadata = {
  title: "Jarvis",
  description: "Assistente pessoal Jarvis",
};

export default async function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  await headers(); // Read auth configuration at request time, not at image build time.
  return (
    <html lang="pt-BR">
      <body>
        {process.env.JARVIS_AUTH_REQUIRED === "true" && <AuthControls />}
        {children}
      </body>
    </html>
  );
}
