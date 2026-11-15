import type { Metadata } from "next";
import { SectionPlaceholder } from "@/components/seller/section-placeholder";

export const metadata: Metadata = {
  title: "Склад",
};

export default function InventoryPage() {
  return (
    <SectionPlaceholder
      title="Склад"
      description="Здесь появятся остатки товаров и уведомления о низком запасе."
    />
  );
}
