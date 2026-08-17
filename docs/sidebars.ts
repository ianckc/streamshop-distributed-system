import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

const sidebars: SidebarsConfig = {
  docsSidebar: [
    'intro',
    {
      type: 'category',
      label: 'Getting Started',
      items: ['getting-started/quick-start'],
    },
    {
      type: 'category',
      label: 'Services',
      items: ['services/order-api'],
    },
    {
      type: 'category',
      label: 'Architecture',
      items: [
        'architecture/overview',
        'architecture/api-gateway',
        'architecture/load-balancing',
        'architecture/caching',
        'architecture/messaging',
        'architecture/microservices',
        'architecture/storage',
        'architecture/observability',
      ],
    },
    {
      type: 'category',
      label: 'Data Flows',
      items: ['data-flows/order-placement'],
    },
    {
      type: 'category',
      label: 'Architecture Decision Records',
      items: [
        'adr/compose-over-k8s',
        'adr/redpanda-over-kafka',
        'adr/traefik-and-nginx',
        'adr/clickhouse-for-olap',
        'adr/mongo-for-catalog',
        'adr/at-least-once-delivery',
      ],
    },
    {
      type: 'category',
      label: 'Runbooks',
      items: [
        'runbooks/reset-data',
        'runbooks/inspect-kafka-lag',
        'runbooks/add-catalog-replica',
      ],
    },
    {
      type: 'category',
      label: 'Development',
      items: ['development/local-setup'],
    },
  ],
};

export default sidebars;
