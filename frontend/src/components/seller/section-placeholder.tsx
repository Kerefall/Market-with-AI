type SectionPlaceholderProps = {
  title: string;
  description: string;
};

export function SectionPlaceholder({ title, description }: SectionPlaceholderProps) {
  return (
    <>
      <h1 className="text-xl font-semibold">{title}</h1>
      <p className="mt-3 text-gray-600">{description}</p>
    </>
  );
}
