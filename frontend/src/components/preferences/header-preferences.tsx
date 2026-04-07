import { Languages, Monitor, MoonStar, SunMedium } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuLabel,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useTheme } from "@/components/providers/theme-provider";
import { changeAppLanguage } from "@/lib/i18n";
import { supportedLanguages, type SupportedLanguage, type ThemeMode } from "@/lib/preferences";

function ThemeIcon({ theme }: { theme: ThemeMode }) {
  if (theme === "light") {
    return <SunMedium className="size-4" />;
  }

  if (theme === "dark") {
    return <MoonStar className="size-4" />;
  }

  return <Monitor className="size-4" />;
}

export function HeaderPreferences() {
  const { t, i18n } = useTranslation();
  const { theme, setTheme } = useTheme();

  const handleLanguageChange = (value: string) => {
    void changeAppLanguage(value as SupportedLanguage).catch((error) => {
      console.error("Failed to change application language", error);
    });
  };

  return (
    <div className="flex items-center gap-1.5">
      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            aria-label={t("common.theme")}
            size="icon-sm"
            title={t("common.theme")}
            variant="ghost"
          >
            <ThemeIcon theme={theme} />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-40">
          <DropdownMenuLabel>{t("common.theme")}</DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuRadioGroup onValueChange={(value) => setTheme(value as ThemeMode)} value={theme}>
            <DropdownMenuRadioItem value="light">{t("common.light")}</DropdownMenuRadioItem>
            <DropdownMenuRadioItem value="dark">{t("common.dark")}</DropdownMenuRadioItem>
            <DropdownMenuRadioItem value="system">{t("common.system")}</DropdownMenuRadioItem>
          </DropdownMenuRadioGroup>
        </DropdownMenuContent>
      </DropdownMenu>

      <DropdownMenu>
        <DropdownMenuTrigger asChild>
          <Button
            aria-label={t("common.language")}
            size="icon-sm"
            title={t("common.language")}
            variant="ghost"
          >
            <Languages className="size-4" />
          </Button>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end" className="w-44">
          <DropdownMenuLabel>{t("common.language")}</DropdownMenuLabel>
          <DropdownMenuSeparator />
          <DropdownMenuRadioGroup
            onValueChange={handleLanguageChange}
            value={i18n.resolvedLanguage ?? i18n.language}
          >
            {supportedLanguages.map((language) => (
              <DropdownMenuRadioItem key={language} value={language}>
                {language === "zh-CN" ? t("common.simplifiedChinese") : t("common.english")}
              </DropdownMenuRadioItem>
            ))}
          </DropdownMenuRadioGroup>
        </DropdownMenuContent>
      </DropdownMenu>
    </div>
  );
}
