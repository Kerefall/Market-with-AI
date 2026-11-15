import type { Metadata } from "next";
import { SectionPlaceholder } from "@/components/seller/section-placeholder";

export const metadata: Metadata = {
  title: "Аналитика",
};

export default function AnalyticsPage() {
  return (
    <SectionPlaceholder
      title="Аналитика"
      description="Здесь появятся выручка, средний чек, воронка и топ товаров."
    />
  );
}
