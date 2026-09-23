import { useCallback, useEffect, useState } from 'react';
import { Badge, Button, Card, Col, Empty, Row, Select, Space, Spin, Statistic, Tag, Typography, message } from 'antd';
import { ThunderboltOutlined } from '@ant-design/icons';
import type { Alert, Device, Greenhouse, Reading } from '../types/domain';
import { getGreenhouse, listGreenhouses, simulate } from '../api/greenhouse';
import { getAlerts, getDevices, getLatest, login } from '../api/monitoring';
import MetricCards from '../components/dashboard/MetricCards';
import SensorStatusList from '../components/dashboard/SensorStatusList';
import TrendChart from '../components/charts/TrendChart';
import DevicePanel from '../components/dashboard/DevicePanel';
import AlertList from '../components/dashboard/AlertList';
import { useWebSocket } from '../hooks/useWebSocket';

const { Title, Text } = Typography;

export default function DashboardPage() {
  const [greenhouses, setGreenhouses] = useState<Greenhouse[]>([]);
  const [selected, setSelected] = useState<number>();
  const [detail, setDetail] = useState<Greenhouse>();
  const [readings, setReadings] = useState<Reading[]>([]);
  const [devices, setDevices] = useState<Device[]>([]);
  const [alerts, setAlerts] = useState<Alert[]>([]);
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(async (id = selected) => {
    if (!id) return;
    try {
      const [info, latest, deviceRows, alertRows] = await Promise.all([getGreenhouse(id), getLatest(id), getDevices(id), getAlerts(id)]);
      setDetail(info);
      setReadings(latest);
      setDevices(deviceRows);
      setAlerts(alertRows);
    } catch {
      message.error('监测数据加载失败');
    }
  }, [selected]);

  useEffect(() => {
    (async () => {
      try {
        if (!localStorage.getItem('token')) {
          const auth = await login();
          localStorage.setItem('token', auth.token);
        }
        const rows = await listGreenhouses();
        setGreenhouses(rows);
        setSelected(rows[0]?.id);
      } catch {
        message.error('无法连接后端服务');
      } finally {
        setLoading(false);
      }
    })();
  }, []);

  useEffect(() => {
    void refresh();
    const timer = window.setInterval(() => void refresh(), 30000);
    return () => window.clearInterval(timer);
  }, [refresh]);

  useWebSocket(useCallback(() => void refresh(), [refresh]));

  const runSimulation = async () => {
    if (!selected) return;
    try {
      await simulate(selected);
      message.success('已生成一轮模拟采样');
      await refresh();
    } catch {
      message.error('模拟数据生成失败');
    }
  };

  if (loading) return <Spin size="large" />;
  if (!greenhouses.length) return <Empty description="暂无温室" />;

  const sensors = detail?.sensors ?? [];
  const offlineCount = sensors.filter((item) => item.status === 'offline').length;

  return (
    <Space direction="vertical" size={20} style={{ width: '100%' }}>
      <div className="page-heading">
        <div>
          <Title level={2}>温室环境总览</Title>
          <Text type="secondary">每 30 秒自动刷新 · 传感器 5 分钟未上报自动离线 · WebSocket 实时推送</Text>
        </div>
        <Space>
          <Select
            aria-label="选择温室"
            value={selected}
            style={{ width: 260 }}
            options={greenhouses.map((item) => {
              const count = item.sensors.filter((sensor) => sensor.status === 'offline').length;
              return {
                value: item.id,
                label: (
                  <span>
                    {item.name}
                    {count > 0 && <Tag color="error" style={{ marginInlineStart: 8 }}>{count} 个离线</Tag>}
                  </span>
                ),
              };
            })}
            onChange={setSelected}
          />
          <Button type="primary" icon={<ThunderboltOutlined />} onClick={runSimulation}>生成模拟数据</Button>
        </Space>
      </div>
      <Card
        className="greenhouse-summary"
        title={detail?.name ?? '温室'}
        extra={<Text>{detail?.location} · {detail?.area}㎡</Text>}
      >
        <Row gutter={[24, 16]} align="middle" justify="space-between">
          <Col flex="auto">
            <Space size="large" wrap>
              <Statistic title="传感器总数" value={sensors.length} />
              <Statistic title="在线传感器" value={sensors.length - offlineCount} valueStyle={{ color: '#3f8600' }} />
              <Badge offset={[6, -2]} status={offlineCount ? 'error' : 'success'} text="">
                <Statistic title="离线传感器" value={offlineCount} valueStyle={offlineCount ? { color: '#cf1322' } : undefined} />
              </Badge>
            </Space>
          </Col>
          <Col>
            <Text type="secondary">{offlineCount > 0 ? '存在拔线或失联传感器，旧读数不再作为当前值展示' : '全部传感器在线'}</Text>
          </Col>
        </Row>
        <div className="metric-cards-wrap">
          <MetricCards sensors={sensors} readings={readings} />
        </div>
      </Card>
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={15}>
          <Card title="24 小时环境趋势（仅在线传感器当前读数）" extra={<Button size="small" onClick={() => void refresh()}>立即刷新</Button>}>
            <TrendChart readings={readings.filter((row) => row.sensor?.status === 'online')} />
          </Card>
        </Col>
        <Col xs={24} lg={9}>
          <DevicePanel devices={devices} onUpdated={() => void refresh()} />
        </Col>
        <Col xs={24}>
          <SensorStatusList sensors={sensors} />
        </Col>
        <Col xs={24}>
          <AlertList alerts={alerts} onUpdated={() => void refresh()} />
        </Col>
      </Row>
    </Space>
  );
}
