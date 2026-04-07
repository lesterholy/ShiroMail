import { ConsolePlaceholderPage } from "../../../components/layout/console-placeholder-page";

type UserSectionPageProps = {
  title: string;
  description: string;
  action: string;
};

export function UserSectionPage({ title, description, action }: UserSectionPageProps) {
  return (
    <ConsolePlaceholderPage
      description={description}
      eyebrow="User section"
      primaryAction={action}
      title={title}
    />
  );
}
