import { useEffect, useMemo, useState } from "react";
import {
  Combobox,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxInput,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox";
import { cn } from "@/lib/utils";

export type OptionComboboxOption = {
  value: string;
  label: string;
  keywords?: string[];
  disabled?: boolean;
};

type OptionComboboxProps = {
  value?: string;
  onValueChange: (value: string) => void;
  options: OptionComboboxOption[];
  placeholder: string;
  searchPlaceholder: string;
  emptyLabel: string;
  ariaLabel: string;
  className?: string;
  contentClassName?: string;
  disabled?: boolean;
};

export function OptionCombobox({
  value,
  onValueChange,
  options,
  placeholder,
  searchPlaceholder,
  emptyLabel,
  ariaLabel,
  className,
  contentClassName,
  disabled = false,
}: OptionComboboxProps) {
  const selectedOption = useMemo(
    () => options.find((option) => option.value === value) ?? null,
    [options, value],
  );
  const [open, setOpen] = useState(false);
  const [inputValue, setInputValue] = useState(selectedOption?.label ?? "");

  useEffect(() => {
    setInputValue(selectedOption?.label ?? "");
  }, [selectedOption]);

  return (
    <Combobox
      autoHighlight
      items={options}
      itemToStringValue={(item: OptionComboboxOption) =>
        [item.label, item.value, ...(item.keywords ?? [])].join(" ")
      }
      inputValue={inputValue}
      open={open}
      openOnInputClick
      value={selectedOption}
      onInputValueChange={(nextInputValue) => {
        setInputValue(nextInputValue);
        if (!disabled) {
          setOpen(true);
        }
      }}
      onOpenChange={setOpen}
      onValueChange={(nextValue) => {
        onValueChange(nextValue?.value ?? "");
        setInputValue(nextValue?.label ?? "");
        setOpen(false);
      }}
    >
      <ComboboxInput
        aria-label={ariaLabel}
        className={cn("w-full", className)}
        disabled={disabled}
        onClick={() => {
          if (!disabled) {
            setOpen(true);
          }
        }}
        placeholder={selectedOption ? searchPlaceholder : placeholder}
        showTrigger={false}
      />
      <ComboboxContent className={contentClassName}>
        <ComboboxEmpty>{emptyLabel}</ComboboxEmpty>
        <ComboboxList>
          {(item: OptionComboboxOption) => (
            <ComboboxItem disabled={item.disabled} key={item.value} value={item}>
              {item.label}
            </ComboboxItem>
          )}
        </ComboboxList>
      </ComboboxContent>
    </Combobox>
  );
}
