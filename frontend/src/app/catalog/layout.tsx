import type { ReactNode } from "react";
import { CatalogHeader } from "@/components/catalog/catalog-header";

export default function CatalogLayout({ children }: { children: ReactNode }) {
  return (
    <>
      <CatalogHeader />
      <main className="mx-auto max-w-6xl p-6">{children}</main>
    </>
  );
}
