export type PaginatedItems<T> = {
  page: number;
  totalPages: number;
  total: number;
  items: T[];
};

export function paginateItems<T>(items: T[], page: number, pageSize: number): PaginatedItems<T> {
  const totalPages = Math.max(1, Math.ceil(items.length / pageSize));
  const safePage = Math.min(Math.max(page, 1), totalPages);
  const start = (safePage - 1) * pageSize;

  return {
    page: safePage,
    totalPages,
    total: items.length,
    items: items.slice(start, start + pageSize),
  };
}
