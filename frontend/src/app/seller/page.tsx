import type { Metadata } from "next";
import { StoreProfileForm } from "@/components/seller/store-profile";

export const metadata: Metadata = { title: "Профиль магазина" };

export default function StorePage() {
  return (
    <div className="space-y-5">
      <h1 className="text-2xl font-semibold">Профиль магазина</h1>
      <StoreProfileForm />
    </div>
  );
}
