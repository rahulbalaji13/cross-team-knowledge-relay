import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Cross-Team Knowledge Relay",
  description: "Post bounties and match experts across teams.",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en">
      <body className="antialiased">{children}</body>
    </html>
  );
}
