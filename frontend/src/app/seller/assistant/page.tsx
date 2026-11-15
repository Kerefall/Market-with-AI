import type { Metadata } from "next";
import { SectionPlaceholder } from "@/components/seller/section-placeholder";

export const metadata: Metadata = {
  title: "ИИ-помощник",
};

export default function AssistantPage() {
  return (
    <SectionPlaceholder
      title="ИИ-помощник"
      description="Здесь появятся создание карточки по фото, тексты и SEO, проверка карточки, подсказка цены, аналитика словами и чат-помощник."
    />
  );
}
