import type { Metadata } from "next";
import "./globals.css";
import { themeInitializationScript } from "./theme";

export const metadata: Metadata = {
  title: "Rethra — Yanıtları bulmanıza ve bilgiyi uygulamaya koymanıza yardımcı olur",
  description: "Rethra, açık kaynak bir bilgi çerçevesidir; RAG soru-cevap, Agent muhakemesi ve otomatik Wiki'yi tek yerde birleştirir. v0.8.2 ile ajan yerel tarayıcıyı kullanabilir, bilgi tabanı MCP Server olarak dışarıya sunulabilir ve sohbetler dallandırılıp geri alınabilir; şirket içi dağıtım desteklenir.",
  // Reuse the documentation favicon so the homepage adds nothing at the site root.
  icons: { icon: "/docs/favicon.ico" },
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="tr" className="h-full antialiased" suppressHydrationWarning>
      <head><script dangerouslySetInnerHTML={{ __html: themeInitializationScript }} /></head>
      <body className="min-h-full flex flex-col">{children}</body>
    </html>
  );
}
