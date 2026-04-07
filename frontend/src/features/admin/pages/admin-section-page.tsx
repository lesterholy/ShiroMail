import { ConsolePlaceholderPage } from "../../../components/layout/console-placeholder-page";

type AdminSectionPageProps = {
  title: string;
  description: string;
  action: string;
};

export function AdminSectionPage({ title, description, action }: AdminSectionPageProps) {
  return (
    <ConsolePlaceholderPage
      description={description}
      eyebrow="Admin section"
      primaryAction={action}
      title={title}
    />
  );
}
