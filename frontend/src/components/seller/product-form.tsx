"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState, type ChangeEvent, type FormEvent } from "react";
import { ProductPhoto } from "@/components/catalog/product-photo";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { categories, saveProduct, useDemoStore, type Product } from "@/lib/demo-store";

const selectClass = "h-8 w-full rounded-lg border bg-background px-2.5 text-sm";

export function ProductForm({ product }: { product?: Product }) {
  const router = useRouter();
  const [name, setName] = useState(product?.name ?? "");
  const [description, setDescription] = useState(product?.description ?? "");
  const [photo, setPhoto] = useState(product?.photo ?? "");
  const [readingPhoto, setReadingPhoto] = useState(false);
  const [priceSuggestion, setPriceSuggestion] = useState(false);
  const [error, setError] = useState("");

  function changePhoto(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) return;
    event.target.value = "";
    setError("");
    if (!["image/png", "image/jpeg", "image/webp"].includes(file.type) || file.size > 750_000) {
      setError("Выберите PNG, JPG или WebP размером до 750 КБ.");
      return;
    }
    setReadingPhoto(true);
    const reader = new FileReader();
    reader.onload = () => {
      setPhoto(String(reader.result));
      setReadingPhoto(false);
    };
    reader.onerror = () => {
      setError("Не удалось загрузить фото. Попробуйте ещё раз.");
      setReadingPhoto(false);
    };
    reader.readAsDataURL(file);
  }

  function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    if (!name.trim()) {
      setError("Укажите название товара.");
      return;
    }
    const data = new FormData(event.currentTarget);
    const value = (key: string) => String(data.get(key) ?? "").trim();
    const price = Number(value("price"));
    const stock = Number(value("stock"));
    if (!Number.isFinite(price) || price < 0 || !Number.isSafeInteger(stock) || stock < 0) {
      setError("Проверьте цену и остаток.");
      return;
    }

    const submitter = (event.nativeEvent as SubmitEvent).submitter;
    const status = (submitter?.getAttribute("value") ?? product?.status) === "published" ? "published" : "draft";
    try {
      saveProduct({
        id: product?.id ?? crypto.randomUUID(),
        name: name.trim(),
        description: description.trim(),
        category: value("category"),
        details: value("details"),
        variants: value("variants"),
        tags: value("tags"),
        price,
        stock,
        photo,
        status,
      });
    } catch {
      setError("Не удалось сохранить товар в браузере. Попробуйте уменьшить размер фото.");
      return;
    }
    router.push("/seller/products");
  }

  return (
    <div className="max-w-4xl space-y-4">
      <Link href="/seller/products" className="text-sm underline">К списку товаров</Link>
      <h1 className="text-2xl font-semibold">{product ? "Редактирование товара" : "Новый товар"}</h1>
      <form onSubmit={submit} className="grid gap-6 md:grid-cols-[minmax(0,1fr)_minmax(0,2fr)]">
        <div className="space-y-3">
          <ProductPhoto photo={photo} name={name || "Фото товара"} />
          <label className="block space-y-1 text-sm">
            <span>Фото товара</span>
            <Input type="file" accept="image/jpeg,image/png,image/webp" onChange={changePhoto} disabled={readingPhoto} />
          </label>
          {photo && <Button type="button" variant="outline" onClick={() => setPhoto("")}>Убрать фото</Button>}
        </div>
        <div className="space-y-4">
          <label className="block space-y-1 text-sm">
            <span>Название</span>
            <Input required maxLength={120} value={name} onChange={(event) => setName(event.target.value)} />
          </label>
          <label className="block space-y-1 text-sm">
            <span>Категория</span>
            <select name="category" className={selectClass} defaultValue={product?.category}>
              {categories.map((category) => <option key={category}>{category}</option>)}
            </select>
          </label>
          <div className="space-y-1 text-sm">
            <label htmlFor="product-description">Описание</label>
            <Textarea id="product-description" maxLength={2000} value={description} onChange={(event) => setDescription(event.target.value)} />
            <Button
              type="button"
              variant="outline"
              disabled={!name.trim()}
              onClick={() => setDescription("Товар «" + name.trim() + "» подходит для повседневного использования.")}
            >
              Предложить описание (демо)
            </Button>
          </div>
          <label className="block space-y-1 text-sm">
            <span>Характеристики</span>
            <Textarea name="details" defaultValue={product?.details} maxLength={1000} />
          </label>
          <div className="grid grid-cols-2 gap-3">
            <label className="space-y-1 text-sm">
              <span>Цена, ₽</span>
              <Input name="price" type="number" min="0" max="100000000" step="0.01" defaultValue={product?.price} required />
            </label>
            <label className="space-y-1 text-sm">
              <span>Остаток</span>
              <Input name="stock" type="number" min="0" max="1000000" step="1" defaultValue={product?.stock ?? 0} required />
            </label>
          </div>
          <div className="space-y-1">
            <Button type="button" variant="outline" onClick={() => setPriceSuggestion(true)}>Подсказать цену (демо)</Button>
            {priceSuggestion && <p role="status" className="text-sm text-muted-foreground">Пример рекомендации: от 1 100 до 1 500 ₽</p>}
          </div>
          <label className="block space-y-1 text-sm">
            <span>Варианты</span>
            <Input name="variants" defaultValue={product?.variants} maxLength={300} placeholder="Через запятую" />
          </label>
          <label className="block space-y-1 text-sm">
            <span>Теги для поиска</span>
            <Input name="tags" defaultValue={product?.tags} maxLength={300} placeholder="Через запятую" />
          </label>
          {error && <p role="alert" className="text-sm text-destructive">{error}</p>}
          <div className="flex flex-wrap gap-2">
            <Button type="submit" value={product?.status ?? "draft"} disabled={readingPhoto}>Сохранить</Button>
            {product?.status !== "published" && (
              <Button type="submit" value="published" variant="outline" disabled={readingPhoto}>Опубликовать на витрине</Button>
            )}
            {product?.status === "published" && (
              <Button type="submit" value="draft" variant="outline" disabled={readingPhoto}>Вернуть в черновики</Button>
            )}
          </div>
        </div>
      </form>
    </div>
  );
}

export function EditProduct({ id }: { id: string }) {
  const { products, ready } = useDemoStore();
  const product = products.find((item) => item.id === id);
  if (!ready) return <p role="status">Загрузка товара…</p>;
  if (product) return <ProductForm key={id} product={product} />;
  return (
    <div className="space-y-3">
      <h1 className="text-2xl font-semibold">Товар не найден</h1>
      <Link href="/seller/products" className="underline">К списку товаров</Link>
    </div>
  );
}
