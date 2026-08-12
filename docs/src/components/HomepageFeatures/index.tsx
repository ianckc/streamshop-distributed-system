import type {ReactNode} from 'react';
import clsx from 'clsx';
import Heading from '@theme/Heading';
import styles from './styles.module.css';

type FeatureItem = {
  title: string;
  description: ReactNode;
};

const FeatureList: FeatureItem[] = [
  {
    title: 'Polyglot Microservices',
    description: (
      <>
        Four services in Rust, Go, Python, and Node.js — each with a distinct
        responsibility, not four identical CRUD apps.
      </>
    ),
  },
  {
    title: 'Multi-Store Architecture',
    description: (
      <>
        PostgreSQL for transactions, MongoDB for catalog, ClickHouse for
        analytics, MinIO for objects, and Redis for caching.
      </>
    ),
  },
  {
    title: 'Event-Driven Pipeline',
    description: (
      <>
        Orders publish to Redpanda (Kafka-compatible). A Rust consumer
        enriches events and feeds the OLAP store asynchronously.
      </>
    ),
  },
  {
    title: 'Production Patterns',
    description: (
      <>
        API gateway, load balancing, cache-aside, idempotency keys,
        at-least-once delivery, and distributed tracing with OpenTelemetry.
      </>
    ),
  },
  {
    title: 'Runs Locally',
    description: (
      <>
        Everything boots with <code>docker compose up</code>. No cloud
        account or credit card required.
      </>
    ),
  },
  {
    title: 'Built to Showcase',
    description: (
      <>
        Architecture docs, ADRs, runbooks, and sequence diagrams — designed
        for engineers reviewing a portfolio project.
      </>
    ),
  },
];

function Feature({title, description}: FeatureItem) {
  return (
    <div className={clsx('col col--4', styles.feature)}>
      <Heading as="h3">{title}</Heading>
      <p>{description}</p>
    </div>
  );
}

export default function HomepageFeatures(): ReactNode {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className="row">
          {FeatureList.map((props, idx) => (
            <Feature key={idx} {...props} />
          ))}
        </div>
      </div>
    </section>
  );
}
