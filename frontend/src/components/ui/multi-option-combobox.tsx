import { useMemo, useRef, useState } from "react";
import {
  Combobox,
  ComboboxChip,
  ComboboxChips,
  ComboboxChipsInput,
  ComboboxContent,
  ComboboxEmpty,
  ComboboxItem,
  ComboboxList,
} from "@/components/ui/combobox";
import type { OptionComboboxOption } from "@/components/ui/option-combobox";
import { cn } from "@/lib/utils";

type MultiOptionComboboxProps = {
  values?: string[];
  onValuesChange: (values: string[]) => void;
  options: readonly OptionComboboxOption[];
  placeholder: string;
  searchPlaceholder: string;
  emptyLabel: string;
  ariaLabel: string;
  className?: string;
  contentClassName?: string;
  disabled?: boolean;
};

export function MultiOptionCombobox({
  values = [],
  onValuesChange,
  options,
  placeholder,
  searchPlaceholder,
  emptyLabel,
  ariaLabel,
  className,
  contentClassName,
  disabled = false,
}: MultiOptionComboboxProps) {
  const [open, setOpen] = useState(false);
  const [inputValue, setInputValue] = useState("");
  const anchorRef = useRef<HTMLDivElement | null>(null);

  const selectedOptions = useMemo(() => {
    const selectedSet = new Set(values);
    return options.filter((option) => selectedSet.has(option.value));
  }, [options, values]);

  return (
    <Combobox
      multiple
      autoHighlight
      items={options}
      itemToStringLabel={(item: OptionComboboxOption) => item.label}
      itemToStringValue={(item: OptionComboboxOption) =>
        [item.label, item.value, ...(item.keywords ?? [])].join(" ")
      }
      isItemEqualToValue={(item: OptionComboboxOption, value: OptionComboboxOption) =>
        item.value === value.value
      }
      inputValue={inputValue}
      open={open}
      openOnInputClick
      value={selectedOptions}
      onInputValueChange={(nextInputValue) => {
        setInputValue(nextInputValue);
        if (!disabled) {
          setOpen(true);
        }
      }}
      onOpenChange={setOpen}
      onValueChange={(nextValue) => {
        onValuesChange((nextValue ?? []).map((item) => item.value));
        setInputValue("");
        if (!disabled) {
          setOpen(true);
        }
      }}
    >
      <ComboboxChips
        ref={anchorRef}
        className={cn("w-full", className)}
        onClick={() => {
          if (!disabled) {
            setOpen(true);
          }
        }}
      >
        {selectedOptions.map((option) => (
          <ComboboxChip key={option.value}>{option.label}</ComboboxChip>
        ))}
        <ComboboxChipsInput
          aria-label={ariaLabel}
          className="min-w-24"
          disabled={disabled}
          onFocus={() => {
            if (!disabled) {
              setOpen(true);
            }
          }}
          placeholder={selectedOptions.length > 0 ? searchPlaceholder : placeholder}
        />
      </ComboboxChips>
      <ComboboxContent anchor={anchorRef} className={contentClassName}>
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
