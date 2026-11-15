import type { Metadata } from "next";
import { SectionPlaceholder } from "@/components/seller/section-placeholder";

export const metadata: Metadata = {
  title: "Продвижение",
};

export default function PromotionsPage() {
  return (
    <SectionPlaceholder
      title="Продвижение"
      description="Здесь появятся скидки, промокоды, акции и отзывы."
    />
  );
}
