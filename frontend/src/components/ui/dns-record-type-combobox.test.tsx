import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { DNSRecordTypeCombobox } from "./dns-record-type-combobox";

describe("DNSRecordTypeCombobox", () => {
  it("shows the current record type and lets the user pick another one", async () => {
    const onValueChange = vi.fn();

    render(<DNSRecordTypeCombobox value="TXT" onValueChange={onValueChange} />);

    const combobox = screen.getByRole("combobox", { name: "记录类型" });
    expect(combobox).toHaveValue("TXT");

    fireEvent.click(screen.getByRole("button", { name: "" }));
    fireEvent.click(await screen.findByRole("option", { name: "A" }));

    expect(onValueChange).toHaveBeenCalledWith("A");
  });
});
