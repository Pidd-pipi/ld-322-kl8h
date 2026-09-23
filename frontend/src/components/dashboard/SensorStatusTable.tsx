import { Card, Table, Tag, Tooltip } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import type { Reading, Sensor } from '../../types/domain';
import StatusTag from '../common/StatusTag';
import { formatAgo, formatReportedAt, isOnline, SENSOR_LABELS } from '../../utils/sensorStatus';

type Props = { sensors: Sensor[]; readings: Reading[]; now: number };

export default function SensorStatusTable({ sensors, readings, now }: Props) {
  const latestBySensor = new Map<number, Reading>();
  readings.forEach((row) => latestBySensor.set(row.sensorId, row));

  const columns: ColumnsType<Sensor> = [
    {
      title: '传感器',
      dataIndex: 'name',
      render: (_: string, sensor) => (
        <span>
          <strong>{sensor.name}</strong>
          <span className="sensor-type-hint">{SENSOR_LABELS[sensor.type] ?? sensor.type}</span>
        </span>
      ),
    },
    {
      title: '在线状态',
      dataIndex: 'status',
      width: 110,
      filters: [
        { text: '在线', value: 'online' },
        { text: '离线', value: 'offline' },
      ],
      onFilter: (value, sensor) => (isOnline(sensor, now) ? 'online' : 'offline') === value,
      render: (_: string, sensor) => <StatusTag status={isOnline(sensor, now) ? 'online' : 'offline'} />,
    },
    {
      title: '当前读数',
      width: 150,
      render: (_: unknown, sensor) => {
        const latest = latestBySensor.get(sensor.id);
        if (!isOnline(sensor, now)) {
          return latest ? (
            <Tooltip title={`采集于 ${formatReportedAt(sensor)}，不再代表当前值`}>
              <Tag color="default">无当前值</Tag>
              <span className="stale-hint">旧值 {latest.value.toFixed(1)}{sensor.unit}</span>
            </Tooltip>
          ) : <Tag color="default">无读数</Tag>;
        }
        if (!latest) return <span className="stale-hint">等待上报</span>;
        const outOfRange = sensor.threshold &&
          (latest.value < sensor.threshold.minValue || latest.value > sensor.threshold.maxValue);
        return (
          <span className={outOfRange ? 'reading-abnormal' : 'reading-ok'}>
            {latest.value.toFixed(1)} {sensor.unit}
            {outOfRange && <Tag color="error" style={{ marginLeft: 8 }}>超阈值</Tag>}
          </span>
        );
      },
    },
    {
      title: '阈值范围',
      width: 150,
      render: (_: unknown, sensor) => sensor.threshold
        ? `${sensor.threshold.minValue.toFixed(1)} ~ ${sensor.threshold.maxValue.toFixed(1)} ${sensor.unit}`
        : '—',
    },
    {
      title: '最近上报',
      width: 180,
      render: (_: unknown, sensor) => (
        <Tooltip title={formatReportedAt(sensor)}>
          <span className={isOnline(sensor, now) ? '' : 'stale-hint'}>
            {sensor.lastReportedAt ? formatAgo(sensor, now) : '从未上报'}
          </span>
        </Tooltip>
      ),
    },
  ];

  return (
    <Card title="传感器在线状态" extra={<span className="sensor-type-hint">5 分钟未上报判定为离线</span>}>
      <Table<Sensor>
        rowKey="id"
        size="small"
        columns={columns}
        dataSource={sensors}
        pagination={false}
        rowClassName={(sensor) => (isOnline(sensor, now) ? '' : 'sensor-row-offline')}
      />
    </Card>
  );
}
