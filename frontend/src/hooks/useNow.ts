import { useEffect, useState } from 'react';

// 定时返回当前时间，让五分钟未上报的传感器无需等待接口刷新即可翻为离线。
export function useNow(intervalMs = 30000): number {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const timer = window.setInterval(() => setNow(Date.now()), intervalMs);
    return () => window.clearInterval(timer);
  }, [intervalMs]);
  return now;
}
