import { useMemo, useState } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  WorkspaceBadge,
  WorkspaceEmpty,
  WorkspaceField,
  WorkspacePage,
  WorkspacePanel,
} from "@/components/layout/workspace-ui";
import { NoticeBanner } from "@/components/ui/notice-banner";
import { validateIntegerRange, validateRequiredText } from "@/lib/validation";
import {
  createAdminDoc,
  deleteAdminDoc,
  fetchAdminDocs,
  updateAdminDoc,
} from "../api";
import type { DocArticle } from "../../user/api";

const DEFAULT_DOC_DRAFT = {
  title: "",
  category: "",
  summary: "",
  readTimeMin: 5,
  tagsText: "",
};

export function AdminDocsPage() {
  const queryClient = useQueryClient();
  const [draft, setDraft] = useState(DEFAULT_DOC_DRAFT);
  const [editingDoc, setEditingDoc] = useState<(DocArticle & { tagsText: string }) | null>(null);
  const [pendingDeleteDoc, setPendingDeleteDoc] = useState<DocArticle | null>(null);
  const [formError, setFormError] = useState<string | null>(null);

  const docsQuery = useQuery({ queryKey: ["admin-docs"], queryFn: fetchAdminDocs });

  const refreshDocs = async () => {
    await queryClient.invalidateQueries({ queryKey: ["admin-docs"] });
    await queryClient.invalidateQueries({ queryKey: ["portal-docs"] });
  };

  const createMutation = useMutation({
    mutationFn: createAdminDoc,
    onSuccess: async () => {
      setFormError(null);
      setDraft(DEFAULT_DOC_DRAFT);
      await refreshDocs();
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ docId, input }: { docId: string; input: Omit<DocArticle, "id" | "createdAt" | "updatedAt"> }) =>
      updateAdminDoc(docId, input),
    onSuccess: async () => {
      setFormError(null);
      setEditingDoc(null);
      await refreshDocs();
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteAdminDoc,
    onSuccess: async () => {
      setPendingDeleteDoc(null);
      await refreshDocs();
    },
  });

  const docs = useMemo(() => docsQuery.data ?? [], [docsQuery.data]);
  const canCreate = draft.title.trim() !== "" && draft.category.trim() !== "" && draft.summary.trim() !== "";

  function validateDocInput(input: { title: string; category: string; summary: string; readTimeMin: number }) {
    return (
      validateRequiredText("文档标题", input.title, { minLength: 2, maxLength: 120 }) ||
      validateRequiredText("文档分类", input.category, { minLength: 2, maxLength: 40 }) ||
      validateRequiredText("文档摘要", input.summary, { minLength: 10, maxLength: 2000 }) ||
      validateIntegerRange("阅读时长", input.readTimeMin, { min: 1, max: 240 })
    );
  }

  function handleCreateDoc() {
    const error = validateDocInput(draft);
    if (error) {
      setFormError(error);
      return;
    }
    setFormError(null);
    createMutation.mutate({
      title: draft.title.trim(),
      category: draft.category.trim(),
      summary: draft.summary.trim(),
      readTimeMin: draft.readTimeMin,
      tags: splitTags(draft.tagsText),
    });
  }

  function handleUpdateDoc() {
    if (!editingDoc) {
      return;
    }
    const error = validateDocInput(editingDoc);
    if (error) {
      setFormError(error);
      return;
    }
    setFormError(null);
    updateMutation.mutate({
      docId: editingDoc.id,
      input: {
        title: editingDoc.title.trim(),
        category: editingDoc.category.trim(),
        summary: editingDoc.summary.trim(),
        readTimeMin: editingDoc.readTimeMin,
        tags: splitTags(editingDoc.tagsText),
      },
    });
  }

  const docCards = useMemo(
    () =>
      docs.map((doc) => (
        <Card className="border-border/60 bg-card/85 shadow-none" key={doc.id}>
          <CardContent className="space-y-3 py-4">
            <div className="flex flex-wrap items-center justify-between gap-3">
              <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                <WorkspaceBadge>{doc.category}</WorkspaceBadge>
                <span>{doc.readTimeMin} min</span>
              </div>
              <div className="flex gap-2">
                <Button
                  onClick={() =>
                    setEditingDoc({
                      ...doc,
                      tagsText: (doc.tags ?? []).join(", "),
                    })
                  }
                  size="sm"
                  variant="outline"
                >
                  编辑
                </Button>
                <Button onClick={() => setPendingDeleteDoc(doc)} size="sm" variant="destructive">
                  删除
                </Button>
              </div>
            </div>

            <div className="space-y-1">
              <div className="text-sm font-medium">{doc.title}</div>
              <p className="text-sm leading-6 text-muted-foreground">{doc.summary}</p>
            </div>

            <div className="flex flex-wrap gap-2">
              {(doc.tags ?? []).map((tag) => (
                <WorkspaceBadge key={tag} variant="outline">
                  {tag}
                </WorkspaceBadge>
              ))}
            </div>
          </CardContent>
        </Card>
      )),
    [docs],
  );

  return (
    <WorkspacePage>
      <WorkspacePanel
        description="维护文档中心条目，普通用户与管理员都会读取同一份文档数据。"
        title="文档中心"
      >
        <Card className="border-border/60 bg-muted/10 shadow-none">
          <CardContent className="space-y-4 py-4">
            <div className="grid gap-4 md:grid-cols-2">
              <WorkspaceField label="文档标题">
                <Input
                  onChange={(event) => setDraft((current) => ({ ...current, title: event.target.value }))}
                  placeholder="例如：Webhook 事件"
                  value={draft.title}
                />
              </WorkspaceField>
              <WorkspaceField label="文档分类">
                <Input
                  onChange={(event) => setDraft((current) => ({ ...current, category: event.target.value }))}
                  placeholder="例如：开发文档"
                  value={draft.category}
                />
              </WorkspaceField>
            </div>

            <div className="grid gap-4 md:grid-cols-[0.35fr_0.65fr]">
              <WorkspaceField label="阅读时长（分钟）">
                <Input
                  min="1"
                  onChange={(event) =>
                    setDraft((current) => ({
                      ...current,
                      readTimeMin: Number(event.target.value) || 1,
                    }))
                  }
                  type="number"
                  value={draft.readTimeMin}
                />
              </WorkspaceField>
              <WorkspaceField label="标签">
                <Input
                  onChange={(event) => setDraft((current) => ({ ...current, tagsText: event.target.value }))}
                  placeholder="使用逗号分隔，如 API, Webhook, 鉴权"
                  value={draft.tagsText}
                />
              </WorkspaceField>
            </div>

            <WorkspaceField label="文档摘要">
              <Textarea
                onChange={(event) => setDraft((current) => ({ ...current, summary: event.target.value }))}
                placeholder="输入文档摘要，前台文档中心会直接展示这里的内容。"
                rows={5}
                value={draft.summary}
              />
            </WorkspaceField>

            <div className="flex justify-end">
              <Button disabled={!canCreate || createMutation.isPending} onClick={handleCreateDoc}>
                {createMutation.isPending ? "创建中..." : "新增文档"}
              </Button>
            </div>
            {formError ? <NoticeBanner className="text-sm" onDismiss={() => setFormError(null)} variant="error">{formError}</NoticeBanner> : null}
          </CardContent>
        </Card>

        <Dialog
          onOpenChange={(open) => {
            if (!open) {
              setEditingDoc(null);
              setFormError(null);
            }
          }}
          open={editingDoc !== null}
        >
          <DialogContent className="sm:max-w-2xl">
            <DialogHeader>
              <DialogTitle>编辑文档</DialogTitle>
              <DialogDescription>保存后普通用户文档中心会立即读取更新后的内容。</DialogDescription>
            </DialogHeader>

            <div className="space-y-4">
              <div className="grid gap-4 md:grid-cols-2">
                <WorkspaceField label="文档标题">
                  <Input
                    onChange={(event) =>
                      setEditingDoc((current) =>
                        current ? { ...current, title: event.target.value } : current,
                      )
                    }
                    value={editingDoc?.title ?? ""}
                  />
                </WorkspaceField>
                <WorkspaceField label="文档分类">
                  <Input
                    onChange={(event) =>
                      setEditingDoc((current) =>
                        current ? { ...current, category: event.target.value } : current,
                      )
                    }
                    value={editingDoc?.category ?? ""}
                  />
                </WorkspaceField>
              </div>

              <div className="grid gap-4 md:grid-cols-[0.35fr_0.65fr]">
                <WorkspaceField label="阅读时长（分钟）">
                  <Input
                    min="1"
                    onChange={(event) =>
                      setEditingDoc((current) =>
                        current
                          ? { ...current, readTimeMin: Number(event.target.value) || 1 }
                          : current,
                      )
                    }
                    type="number"
                    value={editingDoc?.readTimeMin ?? 1}
                  />
                </WorkspaceField>
                <WorkspaceField label="标签">
                  <Input
                    onChange={(event) =>
                      setEditingDoc((current) =>
                        current ? { ...current, tagsText: event.target.value } : current,
                      )
                    }
                    value={editingDoc?.tagsText ?? ""}
                  />
                </WorkspaceField>
              </div>

              <WorkspaceField label="文档摘要">
                <Textarea
                  onChange={(event) =>
                    setEditingDoc((current) =>
                      current ? { ...current, summary: event.target.value } : current,
                    )
                  }
                  rows={6}
                  value={editingDoc?.summary ?? ""}
                />
              </WorkspaceField>
            </div>

            <DialogFooter>
              <DialogClose asChild>
                <Button variant="outline">取消</Button>
              </DialogClose>
              <Button
                disabled={!editingDoc || updateMutation.isPending}
                onClick={handleUpdateDoc}
              >
                {updateMutation.isPending ? "保存中..." : "保存修改"}
              </Button>
            </DialogFooter>
            {formError ? <NoticeBanner className="text-sm" onDismiss={() => setFormError(null)} variant="error">{formError}</NoticeBanner> : null}
          </DialogContent>
        </Dialog>

        <AlertDialog
          onOpenChange={(open) => {
            if (!open) {
              setPendingDeleteDoc(null);
            }
          }}
          open={pendingDeleteDoc !== null}
        >
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>删除文档</AlertDialogTitle>
              <AlertDialogDescription>
                删除后普通用户和管理员文档中心都会移除这条文档。
              </AlertDialogDescription>
            </AlertDialogHeader>
            <div className="rounded-lg border border-border/60 bg-muted/10 px-3 py-3 text-sm">
              <div className="font-medium">{pendingDeleteDoc?.title ?? "-"}</div>
              <div className="mt-1 text-muted-foreground">{pendingDeleteDoc?.category ?? "-"}</div>
            </div>
            <AlertDialogFooter>
              <AlertDialogCancel>取消</AlertDialogCancel>
              <AlertDialogAction
                disabled={deleteMutation.isPending}
                onClick={() => pendingDeleteDoc && deleteMutation.mutate(pendingDeleteDoc.id)}
              >
                {deleteMutation.isPending ? "删除中..." : "确认删除"}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>

        {docs.length ? (
          <div className="grid gap-4 lg:grid-cols-2">{docCards}</div>
        ) : (
          <WorkspaceEmpty description="当前还没有可供管理员查看的文档条目。" title="暂无文档内容" />
        )}
      </WorkspacePanel>
    </WorkspacePage>
  );
}

function splitTags(value: string) {
  return value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean);
}
