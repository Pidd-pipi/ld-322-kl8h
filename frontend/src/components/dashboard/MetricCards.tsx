import { Card, Col, Row, Statistic, Tag, Tooltip, Typography } from 'antd';
import { DisconnectOutlined, WarningOutlined } from '@ant-design/icons';
import type { Reading, Sensor } from '../../types/domain';

const labels: Record<string, string> = { temperature: '温度', humidity: '湿度', light: '光照', co2: 'CO₂', soil_moisture: '土壤湿度' };

const formatTime = (value?: string) => (value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '从未上报');

type Props = { sensors: Sensor[]; readings: Reading[] };

export default function MetricCards({ sensors, readings }: Props) {
  const latestBySensor = new Map<number, Reading>();
  readings.forEach((row) => latestBySensor.set(row.sensorId, row));
  return (
    <Row gutter={[16, 16]}>
      {sensors.map((sensor) => {
        const row = latestBySensor.get(sensor.id);
        const online = sensor.status === 'online';
        const abnormal = online && Boolean(row && sensor.threshold && (row.value < sensor.threshold.minValue || row.value > sensor.threshold.maxValue));
        const className = online ? (abnormal ? 'metric-card abnormal' : 'metric-card') : 'metric-card offline';
        const title = (
          <span>
            {labels[sensor.type] ?? sensor.name}
            <Tag color={online ? 'success' : 'error'} style={{ marginLeft: 8 }}>
              {online ? '在线' : '离线'}
            </Tag>
          </span>
        );
        return (
          <Col xs={24} sm={12} xl={8} xxl={4} key={sensor.id}>
            <Card className={className} size="small">
              <div className="metric-card-title">{title}</div>
              {online && row ? (
                <>
                  <Statistic value={row.value} precision={1} suffix={sensor.unit} prefix={abnormal ? <WarningOutlined /> : undefined} />
                  <small>{abnormal ? '超出配置阈值' : '传感器状态正常'} · 更新于 {formatTime(row.recordedAt)}</small>
                </>
              ) : (
                <>
                  <Statistic value="--" suffix={sensor.unit} prefix={<DisconnectOutlined />} />
                  <small>
                    五分钟未上报，无当前读数
                    {row ? (
                      <Tooltip title={`${formatTime(row.recordedAt)} 的读数，不代表当前值`}>
                        <Typography.Link className="stale-value-hint">（最后 {row.value.toFixed(1)}{sensor.unit}）</Typography.Link>
                      </Tooltip>
                    ) : null}
                  </small>
                </>
              )}
            </Card>
          </Col>
        );
      })}
    </Row>
  );
}
