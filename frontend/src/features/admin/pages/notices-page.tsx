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
import { OptionCombobox } from "@/components/ui/option-combobox";
import { Textarea } from "@/components/ui/textarea";
import {
  WorkspaceField,
  WorkspacePage,
  WorkspacePanel,
} from "@/components/layout/workspace-ui";
import {
  createAdminNotice,
  deleteAdminNotice,
  fetchAdminNotices,
  updateAdminNotice,
} from "../api";
import type { NoticeItem } from "../../user/api";
import { formatDateTime } from "../../user/pages/shared";

const DEFAULT_NOTICE_DRAFT = {
  title: "",
  body: "",
  category: "platform",
  level: "info",
};

export function AdminNoticesPage() {
  const queryClient = useQueryClient();
  const [draft, setDraft] = useState(DEFAULT_NOTICE_DRAFT);
  const [editingNotice, setEditingNotice] = useState<NoticeItem | null>(null);
  const [pendingDeleteNotice, setPendingDeleteNotice] = useState<NoticeItem | null>(null);

  const noticesQuery = useQuery({ queryKey: ["admin-notices"], queryFn: fetchAdminNotices });
  const categoryOptions = useMemo(
    () => [
      { value: "platform", label: "platform", keywords: ["system", "platform"] },
      { value: "release", label: "release", keywords: ["publish", "release"] },
      { value: "maintenance", label: "maintenance", keywords: ["ops", "maintenance"] },
    ],
    [],
  );
  const levelOptions = useMemo(
    () => [
      { value: "info", label: "info" },
      { value: "warning", label: "warning" },
    ],
    [],
  );

  const refreshNotices = async () => {
    await queryClient.invalidateQueries({ queryKey: ["admin-notices"] });
  };

  const createMutation = useMutation({
    mutationFn: createAdminNotice,
    onSuccess: async () => {
      setDraft(DEFAULT_NOTICE_DRAFT);
      await refreshNotices();
    },
  });

  const updateMutation = useMutation({
    mutationFn: ({ noticeId, input }: { noticeId: number; input: typeof DEFAULT_NOTICE_DRAFT }) =>
      updateAdminNotice(noticeId, input),
    onSuccess: async () => {
      setEditingNotice(null);
      await refreshNotices();
    },
  });

  const deleteMutation = useMutation({
    mutationFn: deleteAdminNotice,
    onSuccess: async () => {
      setPendingDeleteNotice(null);
      await refreshNotices();
    },
  });

  return (
    <WorkspacePage>
      <WorkspacePanel description="发布、编辑和撤回面向前台用户的公告内容。" title="公告中心">
        <Card className="border-border/60 bg-muted/10 shadow-none">
          <CardContent className="space-y-4 py-4">
            <WorkspaceField label="公告标题">
              <Input
                className="h-9"
                onChange={(event) => setDraft((current) => ({ ...current, title: event.target.value }))}
                placeholder="公告标题"
                value={draft.title}
              />
            </WorkspaceField>

            <div className="grid gap-4 md:grid-cols-2">
              <WorkspaceField label="分类">
                <OptionCombobox
                  ariaLabel="公告分类"
                  emptyLabel="没有匹配的分类"
                  value={draft.category}
                  onValueChange={(value) => setDraft((current) => ({ ...current, category: value }))}
                  options={categoryOptions}
                  placeholder="选择分类"
                  searchPlaceholder="搜索分类"
                />
              </WorkspaceField>

              <WorkspaceField label="级别">
                <OptionCombobox
                  ariaLabel="公告级别"
                  emptyLabel="没有匹配的级别"
                  value={draft.level}
                  onValueChange={(value) => setDraft((current) => ({ ...current, level: value }))}
                  options={levelOptions}
                  placeholder="选择级别"
                  searchPlaceholder="搜索级别"
                />
              </WorkspaceField>
            </div>

            <WorkspaceField label="公告正文">
              <Textarea
                onChange={(event) => setDraft((current) => ({ ...current, body: event.target.value }))}
                placeholder="公告正文"
                rows={5}
                value={draft.body}
              />
            </WorkspaceField>

            <div className="flex justify-end">
              <Button disabled={createMutation.isPending} onClick={() => createMutation.mutate(draft)}>
                {createMutation.isPending ? "发布中..." : "发布公告"}
              </Button>
            </div>
          </CardContent>
        </Card>

        <Dialog onOpenChange={(open) => !open && setEditingNotice(null)} open={editingNotice !== null}>
          <DialogContent className="sm:max-w-2xl">
            <DialogHeader>
              <DialogTitle>编辑公告</DialogTitle>
              <DialogDescription>更新标题、分类、级别和正文，保存后前台会立即读取新内容。</DialogDescription>
            </DialogHeader>

            <div className="space-y-4">
              <WorkspaceField label="公告标题">
                <Input
                  onChange={(event) =>
                    setEditingNotice((current) =>
                      current ? { ...current, title: event.target.value } : current,
                    )
                  }
                  placeholder="公告标题"
                  value={editingNotice?.title ?? ""}
                />
              </WorkspaceField>

              <div className="grid gap-4 md:grid-cols-2">
                <WorkspaceField label="分类">
                  <OptionCombobox
                    ariaLabel="编辑公告分类"
                    emptyLabel="没有匹配的分类"
                    onValueChange={(value) =>
                      setEditingNotice((current) =>
                        current ? { ...current, category: value } : current,
                      )
                    }
                    options={categoryOptions}
                    placeholder="选择分类"
                    searchPlaceholder="搜索分类"
                    value={editingNotice?.category ?? "platform"}
                  />
                </WorkspaceField>

                <WorkspaceField label="级别">
                  <OptionCombobox
                    ariaLabel="编辑公告级别"
                    emptyLabel="没有匹配的级别"
                    onValueChange={(value) =>
                      setEditingNotice((current) =>
                        current ? { ...current, level: value } : current,
                      )
                    }
                    options={levelOptions}
                    placeholder="选择级别"
                    searchPlaceholder="搜索级别"
                    value={editingNotice?.level ?? "info"}
                  />
                </WorkspaceField>
              </div>

              <WorkspaceField label="公告正文">
                <Textarea
                  onChange={(event) =>
                    setEditingNotice((current) =>
                      current ? { ...current, body: event.target.value } : current,
                    )
                  }
                  placeholder="公告正文"
                  rows={6}
                  value={editingNotice?.body ?? ""}
                />
              </WorkspaceField>
            </div>

            <DialogFooter>
              <DialogClose asChild>
                <Button variant="outline">取消</Button>
              </DialogClose>
              <Button
                disabled={!editingNotice || updateMutation.isPending}
                onClick={() => {
                  if (!editingNotice) {
                    return;
                  }
                  updateMutation.mutate({
                    noticeId: editingNotice.id,
                    input: {
                      title: editingNotice.title,
                      body: editingNotice.body,
                      category: editingNotice.category,
                      level: editingNotice.level,
                    },
                  });
                }}
              >
                {updateMutation.isPending ? "保存中..." : "保存修改"}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        <AlertDialog
          onOpenChange={(open) => {
            if (!open) {
              setPendingDeleteNotice(null);
            }
          }}
          open={pendingDeleteNotice !== null}
        >
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>删除公告</AlertDialogTitle>
              <AlertDialogDescription>
                删除后前台将不再显示这条公告，该操作不可撤销。
              </AlertDialogDescription>
            </AlertDialogHeader>
            <div className="rounded-lg border border-border/60 bg-muted/10 px-3 py-3 text-sm">
              <div className="font-medium">{pendingDeleteNotice?.title ?? "-"}</div>
              <div className="mt-1 text-muted-foreground">{pendingDeleteNotice?.category ?? "-"}</div>
            </div>
            <AlertDialogFooter>
              <AlertDialogCancel>取消</AlertDialogCancel>
              <AlertDialogAction
                disabled={deleteMutation.isPending}
                onClick={() => pendingDeleteNotice && deleteMutation.mutate(pendingDeleteNotice.id)}
              >
                {deleteMutation.isPending ? "删除中..." : "确认删除"}
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>

        <div className="space-y-3">
          {(noticesQuery.data ?? []).map((item) => (
            <Card className="border-border/60 bg-card/92 shadow-none" key={item.id}>
              <CardContent className="space-y-3 py-4">
                <div className="flex flex-wrap items-center justify-between gap-3">
                  <div className="flex flex-wrap items-center gap-2 text-xs text-muted-foreground">
                    <span className="rounded-full border border-border/60 px-2 py-1">{item.category}</span>
                    <span className="rounded-full border border-border/60 px-2 py-1">{item.level}</span>
                    <span>{formatDateTime(item.publishedAt)}</span>
                  </div>
                  <div className="flex gap-2">
                    <Button onClick={() => setEditingNotice(item)} size="sm" variant="outline">
                      编辑
                    </Button>
                    <Button onClick={() => setPendingDeleteNotice(item)} size="sm" variant="destructive">
                      删除
                    </Button>
                  </div>
                </div>
                <div className="space-y-1">
                  <div className="text-sm font-medium">{item.title}</div>
                  <p className="text-xs leading-6 text-muted-foreground">{item.body}</p>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      </WorkspacePanel>
    </WorkspacePage>
  );
}
