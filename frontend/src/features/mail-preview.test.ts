import { describe, expect, it } from "vitest";
import { buildMailHtmlPreview, collectInlineCIDTargets } from "./mail-preview";

describe("mail preview cid resolution", () => {
  it("resolves url-encoded cid references in html", () => {
    const preview = buildMailHtmlPreview(
      '<div><img src="cid:ii_l0%40mail.gmail.com" alt="inline" /></div>',
      {
        "ii_l0@mail.gmail.com": "blob:http://localhost/image-1",
      },
    );

    expect(preview.html).toContain('src="blob:http://localhost/image-1"');
    expect(preview.notices).toHaveLength(0);
  });

  it("resolves cid references when parsed content id is already decoded", () => {
    const preview = buildMailHtmlPreview('<img src="cid:logo%40test" alt="inline logo" />', {
      "logo@test": "blob:http://localhost/image-2",
    });

    expect(preview.html).toContain('src="blob:http://localhost/image-2"');
    expect(preview.notices).toHaveLength(0);
  });

  it("collects inline image targets", () => {
    const targets = collectInlineCIDTargets([
      {
        contentId: "<ii_l1@mail.gmail.com>",
        contentType: "image/png",
      },
    ]);

    expect(targets).toHaveLength(1);
    expect(targets[0]).toMatchObject({
      attachmentIndex: 0,
      contentId: "ii_l1@mail.gmail.com",
    });
  });
});
