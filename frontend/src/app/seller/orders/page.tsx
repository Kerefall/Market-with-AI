import type { Metadata } from "next";
import { SectionPlaceholder } from "@/components/seller/section-placeholder";

export const metadata: Metadata = {
  title: "Заказы",
};

export default function OrdersPage() {
  return (
    <SectionPlaceholder
      title="Заказы"
      description="Здесь появятся статусы заказов, фильтры и данные покупателя."
    />
  );
}
