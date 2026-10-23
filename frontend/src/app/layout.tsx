import type { Metadata } from "next";
import { cookies } from "next/headers";
import "./globals.css";
import AccessGate from "./components/AccessGate";
import { ACCESS_COOKIE, isSiteAccessConfigured, verifySiteAccessSession } from "@/lib/site-access";

export const metadata: Metadata = {
  title: "Knowledge Relay | Cross-Team Help",
  description: "Share engineering blockers and find teammates with the skills to help.",
};

export default async function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  const cookieStore = await cookies();
  const hasAccess = verifySiteAccessSession(cookieStore.get(ACCESS_COOKIE)?.value);

  return (
    <html lang="en">
      <body className="antialiased">
        {hasAccess ? children : <AccessGate configured={isSiteAccessConfigured()} />}
      </body>
    </html>
  );
}