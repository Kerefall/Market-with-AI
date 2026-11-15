"use client";

import { useMemo, useSyncExternalStore } from "react";

export const categories = ["Дом", "Одежда", "Подарки", "Красота", "Спорт"];

export type Product = {
  id: string;
  name: string;
  category: string;
  description: string;
  details: string;
  variants: string;
  tags: string;
  price: number;
  stock: number;
  photo: string;
  status: "draft" | "published";
};

export type StoreProfile = {
  name: string;
  description: string;
};

type DemoStore = {
  products: Product[];
  profile: StoreProfile;
};

const emptyDetails = { photo: "", details: "", variants: "", tags: "" };

const initialStore: DemoStore = {
  profile: {
    name: "Лавка",
    description: "Товары для дома, отдыха и подарков.",
  },
  products: [
    {
      ...emptyDetails,
      id: "ceramic-mug",
      name: "Керамическая кружка",
      category: "Дом",
      description: "Кружка с матовой глазурью для чая и кофе.",
      details: "Материал: керамика. Объём: 350 мл.",
      variants: "Молочная, синяя",
      tags: "кружка, керамика, подарок",
      price: 1290,
      stock: 12,
      status: "published",
    },
    {
      ...emptyDetails,
      id: "cotton-bag",
      name: "Хлопковый шопер",
      category: "Одежда",
      description: "Сумка из плотного хлопка с длинными ручками.",
      price: 890,
      stock: 8,
      status: "published",
    },
    {
      ...emptyDetails,
      id: "soy-candle",
      name: "Свеча из соевого воска",
      category: "Подарки",
      description: "Свеча в стеклянной банке с ароматом ванили.",
      price: 1490,
      stock: 5,
      status: "published",
    },
    {
      ...emptyDetails,
      id: "training-band",
      name: "Лента для тренировок",
      category: "Спорт",
      description: "Эластичная лента для разминки и растяжки.",
      price: 690,
      stock: 0,
      status: "draft",
    },
  ],
};

const storageKey = "market-with-ai:demo-store:v1";
const changeEvent = "demo-store-change";

function subscribe(onChange: () => void) {
  function onStorage(event: StorageEvent) {
    if (event.key === storageKey || event.key === null) onChange();
  }

  window.addEventListener("storage", onStorage);
  window.addEventListener(changeEvent, onChange);

  return () => {
    window.removeEventListener("storage", onStorage);
    window.removeEventListener(changeEvent, onChange);
  };
}

function getSnapshot() {
  try {
    return window.localStorage.getItem(storageKey) ?? "";
  } catch {
    return "";
  }
}

function getServerSnapshot() {
  return null;
}

function readStore(raw: string | null): DemoStore {
  try {
    const data = raw ? JSON.parse(raw) : null;
    if (Array.isArray(data?.products) && typeof data.profile?.name === "string") {
      return data;
    }
  } catch {
    // The demo can start from sample data if its local data cannot be read.
  }
  return initialStore;
}

function updateStore(update: (store: DemoStore) => DemoStore) {
  const next = update(readStore(getSnapshot()));
  window.localStorage.setItem(storageKey, JSON.stringify(next));
  window.dispatchEvent(new Event(changeEvent));
}

export function useDemoStore() {
  const raw = useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
  const store = useMemo(() => readStore(raw), [raw]);
  return { ...store, ready: raw !== null };
}

export function saveProduct(product: Product) {
  updateStore((store) => {
    const exists = store.products.some(({ id }) => id === product.id);
    const products = exists
      ? store.products.map((item) => item.id === product.id ? product : item)
      : [...store.products, product];
    return { ...store, products };
  });
}

export function saveStoreProfile(profile: StoreProfile) {
  updateStore((store) => ({ ...store, profile }));
}

const priceFormat = new Intl.NumberFormat("ru-RU", {
  style: "currency",
  currency: "RUB",
  maximumFractionDigits: 2,
  minimumFractionDigits: 0,
});

export function formatPrice(price: number) {
  return priceFormat.format(price);
}
