import { useEffect, useState } from "react";

// usePoll calls fetcher immediately and every intervalMs, keeping the last good
// value. Pass deps to re-subscribe when inputs (e.g. a filter) change.
export function usePoll<T>(
  fetcher: () => Promise<T>,
  initial: T,
  intervalMs = 15000,
  deps: unknown[] = [],
): T {
  const [data, setData] = useState<T>(initial);
  useEffect(() => {
    let alive = true;
    const tick = async () => {
      try {
        const d = await fetcher();
        if (alive) setData(d);
      } catch {
        /* keep last good value */
      }
    };
    tick();
    const id = setInterval(tick, intervalMs);
    return () => {
      alive = false;
      clearInterval(id);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, deps);
  return data;
}
