import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { Dialog, DialogContent, DialogTitle } from "./dialog";
import { OptionCombobox } from "./option-combobox";

function DialogComboboxFixture({ onChange }: { onChange: (value: string) => void }) {
  return (
    <Dialog open>
      <DialogContent>
        <DialogTitle>选择域名</DialogTitle>
        <OptionCombobox
          ariaLabel="域名"
          emptyLabel="没有匹配域名"
          onValueChange={onChange}
          options={[
            { value: "1", label: "alpha.test" },
            { value: "2", label: "beta.test" },
          ]}
          placeholder="选择域名"
          searchPlaceholder="搜索域名"
          value="1"
        />
      </DialogContent>
    </Dialog>
  );
}

describe("OptionCombobox in Dialog", () => {
  it("allows selecting another option inside dialog", async () => {
    const onValueChange = vi.fn();

    render(<DialogComboboxFixture onChange={onValueChange} />);

    fireEvent.click(screen.getByRole("combobox", { name: "域名" }));
    fireEvent.click(await screen.findByRole("option", { name: "beta.test" }));

    expect(onValueChange).toHaveBeenCalledWith("2");
  });
});
