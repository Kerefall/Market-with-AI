import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Маркетплейс с ИИ-помощником",
  description: "Командный проект: маркетплейс с ИИ-помощником для продавцов.",
};

export default function RootLayout({ children }: LayoutProps<"/">) {
  return (
    <html lang="ru">
      <body className="bg-white font-sans text-gray-900">{children}</body>
    </html>
  );
}
