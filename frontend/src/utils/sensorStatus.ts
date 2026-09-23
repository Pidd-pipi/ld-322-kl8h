import type { Sensor } from '../types/domain';

// 与后端 constants.SensorOfflineThreshold 保持一致：五分钟未上报视为离线。
export const OFFLINE_THRESHOLD_MS = 5 * 60 * 1000;

export const SENSOR_LABELS: Record<string, string> = {
  temperature: '温度',
  humidity: '湿度',
  light: '光照',
  co2: 'CO₂',
  soil_moisture: '土壤湿度',
};

export function isOnline(sensor: Sensor, now: number = Date.now()): boolean {
  if (!sensor.lastReportedAt) return false;
  return now - new Date(sensor.lastReportedAt).getTime() <= OFFLINE_THRESHOLD_MS;
}

export function formatReportedAt(sensor: Sensor): string {
  if (!sensor.lastReportedAt) return '从未上报';
  return new Date(sensor.lastReportedAt).toLocaleString('zh-CN', { hour12: false });
}

export function formatAgo(sensor: Sensor, now: number = Date.now()): string {
  if (!sensor.lastReportedAt) return '';
  const seconds = Math.max(0, Math.floor((now - new Date(sensor.lastReportedAt).getTime()) / 1000));
  if (seconds < 60) return `${seconds} 秒前`;
  const minutes = Math.floor(seconds / 60);
  if (minutes < 60) return `${minutes} 分钟前`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours} 小时前`;
  return `${Math.floor(hours / 24)} 天前`;
}
