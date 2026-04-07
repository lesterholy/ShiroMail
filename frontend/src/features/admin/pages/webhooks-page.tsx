import { useState } from "react";
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
import { NoticeBanner } from "@/components/ui/notice-banner";
import { getAPIErrorMessage } from "@/lib/http";
import {
  WorkspaceBadge,
  WorkspaceEmpty,
  WorkspaceField,
  WorkspaceListRow,
  WorkspacePage,
  WorkspacePanel,
} from "@/components/layout/workspace-ui";
import {
  createAdminWebhook,
  fetchAdminWebhooks,
  toggleAdminWebhook,
  updateAdminWebhook,
} from "../api";
import { formatDateTime } from "../../user/pages/shared";

const DEFAULT_EVENTS = ["message.received", "mailbox.released"];

export function AdminWebhooksPage() {
  const queryClient = useQueryClient();
  const [isDialogOpen, setDialogOpen] = useState(false);
  const [editingId, setEditingId] = useState<number | null>(null);
  const [mutationError, setMutationError] = useState<string | null>(null);
  const [actionNotice, setActionNotice] = useState<string | null>(null);
  const [pendingDisableItem, setPendingDisableItem] = useState<
    Awaited<ReturnType<typeof fetchAdminWebhooks>>[number] | null
  >(null);
  const [draft, setDraft] = useState({
    userId: "",
    name: "",
    targetUrl: "",
    events: DEFAULT_EVENTS.join(", "),
  });

  const webhooksQuery = useQuery({
    queryKey: ["admin-webhooks"],
    queryFn: fetchAdminWebhooks,
  });

  const createMutation = useMutation({
    mutationFn: async () => {
      const payload = {
        name: draft.name || "Default webhook",
        targetUrl: draft.targetUrl || "https://sandbox.local/webhooks/default",
        events: draft.events
          .split(",")
          .map((item) => item.trim())
          .filter(Boolean),
      };
      if (editingId) {
        return updateAdminWebhook(editingId, payload);
      }
      return createAdminWebhook({
        userId: Number(draft.userId),
        ...payload,
      });
    },
    onSuccess: async () => {
      setMutationError(null);
      setActionNotice(editingId ? "Webhook 已更新。" : "Webhook 已创建。");
      setDraft({
        userId: "",
        name: "",
        targetUrl: "",
        events: DEFAULT_EVENTS.join(", "),
      });
      setEditingId(null);
      setDialogOpen(false);
      await queryClient.invalidateQueries({ queryKey: ["admin-webhooks"] });
      await queryClient.invalidateQueries({ queryKey: ["admin-overview"] });
    },
    onError: (error) => {
      setMutationError(
        getAPIErrorMessage(
          error,
          editingId
            ? "保存 Webhook 失败，请检查地址、事件和会话状态后重试。"
            : "创建 Webhook 失败，请检查地址、事件和会话状态后重试。",
        ),
      );
    },
  });

  const toggleMutation = useMutation({
    mutationFn: ({ id, enabled }: { id: number; enabled: boolean }) =>
      toggleAdminWebhook(id, enabled),
    onSuccess: async () => {
      setActionNotice("Webhook 状态已更新。");
      await queryClient.invalidateQueries({ queryKey: ["admin-webhooks"] });
      await queryClient.invalidateQueries({ queryKey: ["admin-overview"] });
    },
    onError: (error) => {
      setMutationError(getAPIErrorMessage(error, "切换 Webhook 状态失败，请稍后重试。"));
    },
  });

  const canSubmit = draft.userId.trim() !== "";

  function startCreate() {
    setMutationError(null);
    setActionNotice(null);
    setEditingId(null);
    setDraft({
      userId: "",
      name: "",
      targetUrl: "",
      events: DEFAULT_EVENTS.join(", "),
    });
    setDialogOpen(true);
  }

  function startEdit(item: {
    id: number;
    userId: number;
    name: string;
    targetUrl: string;
    events: string[];
  }) {
    setMutationError(null);
    setActionNotice(null);
    setEditingId(item.id);
    setDraft({
      userId: String(item.userId),
      name: item.name,
      targetUrl: item.targetUrl,
      events: item.events.join(", "),
    });
    setDialogOpen(true);
  }

  return (
    <WorkspacePage>
      <AlertDialog
        open={pendingDisableItem !== null}
        onOpenChange={(open) => {
          if (!open) {
            setPendingDisableItem(null);
          }
        }}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>停用 Webhook？</AlertDialogTitle>
            <AlertDialogDescription>
              {pendingDisableItem
                ? `确认停用 Webhook ${pendingDisableItem.name}？停用后 user #${pendingDisableItem.userId} 的事件将不再投递到 ${pendingDisableItem.targetUrl}。`
                : ""}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>取消</AlertDialogCancel>
            <AlertDialogAction
              onClick={() => {
                if (!pendingDisableItem) {
                  return;
                }
                toggleMutation.mutate({ id: pendingDisableItem.id, enabled: false });
                setPendingDisableItem(null);
              }}
            >
              确认停用
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
      <WorkspacePanel
        action={<Button onClick={startCreate}>新增 Webhook</Button>}
        description="全局查看 webhook 地址、事件和启停状态。"
        title="Webhook"
      >
        {actionNotice ? (
          <NoticeBanner autoHideMs={5000} className="mb-4" onDismiss={() => setActionNotice(null)} variant="success">
            {actionNotice}
          </NoticeBanner>
        ) : null}
        <Dialog
          onOpenChange={(open) => {
            setDialogOpen(open);
            if (open) {
              setMutationError(null);
            }
          }}
          open={isDialogOpen}
        >
          <DialogContent className="sm:max-w-2xl">
            <DialogHeader>
              <DialogTitle>{editingId ? "编辑 Webhook" : "新增 Webhook"}</DialogTitle>
              <DialogDescription>
                {editingId
                  ? "修改目标地址与事件列表，保存后会立即同步到管理员视图。"
                  : "为指定用户创建新的事件回调地址，提交后会立即同步到管理员视图。"}
              </DialogDescription>
            </DialogHeader>

            <div className="grid gap-4 xl:grid-cols-2">
              <WorkspaceField label="所属用户 ID">
                <Input
                  disabled={editingId !== null}
                  min="1"
                  onChange={(event) =>
                    setDraft((current) => ({ ...current, userId: event.target.value }))
                  }
                  placeholder="输入用户 ID"
                  type="number"
                  value={draft.userId}
                />
              </WorkspaceField>
              <WorkspaceField label="Webhook 名称">
                <Input
                  onChange={(event) =>
                    setDraft((current) => ({ ...current, name: event.target.value }))
                  }
                  placeholder="Webhook 名称"
                  value={draft.name}
                />
              </WorkspaceField>
              <WorkspaceField label="回调地址">
                <Input
                  onChange={(event) =>
                    setDraft((current) => ({ ...current, targetUrl: event.target.value }))
                  }
                  placeholder="https://sandbox.local/webhooks/order"
                  value={draft.targetUrl}
                />
              </WorkspaceField>
              <WorkspaceField label="事件列表">
                <Input
                  onChange={(event) =>
                    setDraft((current) => ({ ...current, events: event.target.value }))
                  }
                  placeholder="message.received, mailbox.released"
                  value={draft.events}
                />
              </WorkspaceField>
            </div>

            <DialogFooter>
              {mutationError ? (
                <NoticeBanner autoHideMs={5000} className="mr-auto" onDismiss={() => setMutationError(null)} variant="error">
                  {mutationError}
                </NoticeBanner>
              ) : null}
              <DialogClose asChild>
                <Button variant="outline">取消</Button>
              </DialogClose>
              <Button
                disabled={
                  createMutation.isPending || (editingId === null && !canSubmit)
                }
                onClick={() => createMutation.mutate()}
              >
                {createMutation.isPending
                  ? editingId
                    ? "保存中..."
                    : "创建中..."
                  : editingId
                    ? "保存修改"
                    : "创建 Webhook"}
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>

        {webhooksQuery.data?.length ? (
          <div className="space-y-3">
            {webhooksQuery.data.map((item) => (
              <WorkspaceListRow
                description={
                  <div className="space-y-2">
                    <div className="text-sm text-muted-foreground">{item.targetUrl}</div>
                    <div className="flex flex-wrap gap-1.5 text-[0.8rem] text-muted-foreground">
                      <span>user #{item.userId}</span>
                      {item.events.map((event) => (
                        <WorkspaceBadge key={event} variant="outline">
                          {event}
                        </WorkspaceBadge>
                      ))}
                    </div>
                  </div>
                }
                key={item.id}
                meta={
                  <>
                    <WorkspaceBadge>{item.enabled ? "enabled" : "disabled"}</WorkspaceBadge>
                    <span>{formatDateTime(item.updatedAt)}</span>
                    <Button onClick={() => startEdit(item)} size="sm" variant="secondary">
                      编辑
                    </Button>
                    <Button
                      onClick={() => {
                        if (item.enabled) {
                          setPendingDisableItem(item);
                          return;
                        }
                        toggleMutation.mutate({ id: item.id, enabled: true });
                      }}
                      size="sm"
                      variant="outline"
                    >
                      {item.enabled ? "停用" : "启用"}
                    </Button>
                  </>
                }
                title={item.name}
              />
            ))}
          </div>
        ) : (
          <WorkspaceEmpty
            description="全局 Webhook 建立后，这里会显示目标地址、事件与启停状态。"
            title="暂无 Webhook"
          />
        )}
      </WorkspacePanel>
    </WorkspacePage>
  );
}
