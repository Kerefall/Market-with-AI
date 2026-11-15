import Image from "next/image";
import { ImageIcon } from "lucide-react";

export function ProductPhoto({ photo, name }: { photo: string; name: string }) {
  return (
    <div className="relative flex aspect-4/3 items-center justify-center overflow-hidden rounded-lg bg-muted">
      {photo ? (
        <Image src={photo} alt={name} fill unoptimized className="object-contain" />
      ) : (
        <span className="flex items-center gap-2 text-sm text-muted-foreground">
          <ImageIcon aria-hidden="true" className="size-4" />
          Фото пока нет
        </span>
      )}
    </div>
  );
}
