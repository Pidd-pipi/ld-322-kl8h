import { Card, Col, Row, Statistic, Tag, Tooltip } from 'antd';
import { WarningOutlined, DisconnectOutlined } from '@ant-design/icons';
import type { Reading, Sensor } from '../../types/domain';
import { formatAgo, isOnline, SENSOR_LABELS } from '../../utils/sensorStatus';

type Props = { sensors: Sensor[]; readings: Reading[]; now: number };

export default function MetricCards({ sensors, readings, now }: Props) {
  const latestBySensor = new Map<number, Reading>();
  readings.forEach((row) => latestBySensor.set(row.sensorId, row));

  return (
    <Row gutter={[16, 16]}>
      {sensors.map((sensor) => {
        const online = isOnline(sensor, now);
        const latest = latestBySensor.get(sensor.id);
        const abnormal = online && Boolean(
          latest && sensor.threshold &&
          (latest.value < sensor.threshold.minValue || latest.value > sensor.threshold.maxValue),
        );
        const className = online ? (abnormal ? 'metric-card abnormal' : 'metric-card') : 'metric-card offline';

        return (
          <Col xs={24} sm={12} xl={8} key={sensor.id} className="metric-col">
            <Card className={className}>
              {online ? (
                <Statistic
                  title={SENSOR_LABELS[sensor.type] ?? sensor.name}
                  value={latest ? latest.value : '—'}
                  precision={latest ? 1 : undefined}
                  suffix={latest ? sensor.unit : undefined}
                  prefix={abnormal ? <WarningOutlined /> : undefined}
                />
              ) : (
                <Statistic
                  title={<span>{SENSOR_LABELS[sensor.type] ?? sensor.name} <Tag color="default" icon={<DisconnectOutlined />}>离线</Tag></span>}
                  value="—"
                  suffix={sensor.unit}
                />
              )}
              <small>
                {online ? (
                  abnormal ? '超出配置阈值' : '传感器在线 · 数据正常'
                ) : latest ? (
                  <Tooltip title={`旧读数 ${latest.value.toFixed(1)}${sensor.unit}，采集于 ${new Date(latest.recordedAt).toLocaleString('zh-CN', { hour12: false })}，不再代表当前值`}>
                    <span className="stale-hint">超过 5 分钟未上报 · 旧值 {latest.value.toFixed(1)}{sensor.unit}（仅供参考）</span>
                  </Tooltip>
                ) : (
                  <span className="stale-hint">从未收到上报数据</span>
                )}
              </small>
              {sensor.lastReportedAt && <div className="metric-foot">最近上报：{formatAgo(sensor, now)}</div>}
            </Card>
          </Col>
        );
      })}
    </Row>
  );
}
