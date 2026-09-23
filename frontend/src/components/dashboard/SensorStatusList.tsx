import { Card, Table, Tag } from 'antd';
import type { Sensor } from '../../types/domain';
import type { ColumnsType } from 'antd/es/table';

const labels: Record<string, string> = { temperature: '温度', humidity: '湿度', light: '光照', co2: 'CO₂', soil_moisture: '土壤湿度' };
const formatTime = (value?: string) => (value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '从未上报');

type Row = Sensor & { key: number };

export default function SensorStatusList({ sensors }: { sensors: Sensor[] }) {
  const offlineCount = sensors.filter((item) => item.status === 'offline').length;
  const columns: ColumnsType<Row> = [
    { title: '传感器', dataIndex: 'name', render: (_, row) => `${row.name}（${labels[row.type] ?? row.type}）` },
    {
      title: '在线状态',
      dataIndex: 'status',
      filters: [
        { text: '在线', value: 'online' },
        { text: '离线', value: 'offline' },
      ],
      onFilter: (value, row) => row.status === value,
      render: (status: string) => <Tag color={status === 'online' ? 'success' : 'error'}>{status === 'online' ? '在线' : '离线（超过 5 分钟未上报）'}</Tag>,
    },
    { title: '最近上报时间', dataIndex: 'lastReportedAt', render: (value?: string) => formatTime(value) },
    { title: '阈值范围', dataIndex: 'threshold', render: (_, row) => (row.threshold ? `${row.threshold.minValue} ~ ${row.threshold.maxValue} ${row.unit}` : '—') },
  ];
  return (
    <Card
      className="sensor-status-card"
      title="传感器在线状态"
      extra={
        <span>
          在线 <Tag color="success">{sensors.length - offlineCount}</Tag>
          离线 <Tag color={offlineCount ? 'error' : 'default'}>{offlineCount}</Tag>
        </span>
      }
    >
      <Table<Row>
        size="small"
        pagination={false}
        rowKey="id"
        columns={columns}
        dataSource={sensors.map((item) => ({ ...item, key: item.id }))}
        rowClassName={(row) => (row.status === 'offline' ? 'sensor-row-offline' : '')}
      />
    </Card>
  );
}
