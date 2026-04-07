import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { OptionCombobox } from "./option-combobox";

describe("OptionCombobox", () => {
  it("shows the current option and lets the user pick another one", async () => {
    const onValueChange = vi.fn();

    render(
      <OptionCombobox
        ariaLabel="选择域名"
        emptyLabel="没有匹配域名"
        onValueChange={onValueChange}
        options={[
          { value: "1", label: "alpha.test" },
          { value: "2", label: "beta.test" },
        ]}
        placeholder="选择域名"
        searchPlaceholder="搜索域名"
        value="1"
      />,
    );

    const combobox = screen.getByRole("combobox", { name: "选择域名" });
    expect(combobox).toHaveValue("alpha.test");

    fireEvent.click(combobox);
    fireEvent.click(await screen.findByRole("option", { name: "beta.test" }));

    expect(onValueChange).toHaveBeenCalledWith("2");
  });
});
