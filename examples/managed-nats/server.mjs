import http from 'node:http';
import transport from '@nats-io/transport-node';
import jetstreamModule from '@nats-io/jetstream';

const { connect } = transport;
const { jetstream, jetstreamManager, StorageType } = jetstreamModule;
const port = Number.parseInt(process.env.PORT ?? '8080', 10);
const mode = process.env.NATS_TEST_MODE ?? 'waiting';
const encoder = new TextEncoder();
const decoder = new TextDecoder();

const server = http.createServer((request, response) => {
  response.setHeader('Content-Type', 'application/json');
  if (request.url === '/health') {
    response.end(JSON.stringify({ status: 'ok', natsMode: mode }));
    return;
  }
  response.statusCode = 404;
  response.end(JSON.stringify({ error: 'not found' }));
});

server.listen(port, '0.0.0.0', () => {
  console.log(`HTTP_READY port=${port} mode=${mode}`);
});

if (process.env.NATS_URL && process.env.NATS_TOKEN) {
  runNatsCheck().catch((error) => {
    console.error(`NATS_CHECK_FAILED mode=${mode} message=${error.message}`);
    process.exitCode = 1;
    server.close();
  });
} else {
  console.log('NATS_WAITING credentials_not_attached');
}

async function runNatsCheck() {
  const connection = await connect({
    servers: process.env.NATS_URL,
    token: process.env.NATS_TOKEN,
  });

  try {
    if (mode === 'core') {
      await checkCoreMessaging(connection);
      return;
    }
    if (mode === 'jetstream-write') {
      await writeJetStreamMessage(connection);
      return;
    }
    if (mode === 'jetstream-read') {
      await readJetStreamMessage(connection);
      return;
    }
    throw new Error(`unsupported NATS_TEST_MODE: ${mode}`);
  } finally {
    await connection.drain();
  }
}

async function checkCoreMessaging(connection) {
  const subject = 'guide.core.events';
  const subscription = connection.subscribe(subject, { max: 1 });
  await connection.flush();
  connection.publish(subject, encoder.encode('guide-core-message'));

  const timer = setTimeout(() => subscription.unsubscribe(), 10_000);
  for await (const message of subscription) {
    console.log(`CORE_NATS_OK message=${decoder.decode(message.data)}`);
  }
  clearTimeout(timer);
}

async function writeJetStreamMessage(connection) {
  const manager = await jetstreamManager(connection);
  try {
    await manager.streams.add({
      name: 'GUIDE_EVENTS',
      subjects: ['guide.events'],
      storage: StorageType.File,
      num_replicas: 3,
    });
  } catch (error) {
    if (error.api_error?.err_code !== 10058) {
      throw error;
    }
  }

  const client = jetstream(connection);
  await client.publish(
    'guide.events',
    encoder.encode('guide-persisted-message'),
  );
  const info = await manager.streams.info('GUIDE_EVENTS');
  console.log(`JETSTREAM_WRITE_OK stored=${info.state.messages}`);
}

async function readJetStreamMessage(connection) {
  const manager = await jetstreamManager(connection);
  const message = await manager.streams.getMessage('GUIDE_EVENTS', {
    last_by_subj: 'guide.events',
  });
  console.log(`JETSTREAM_READ_OK message=${decoder.decode(message.data)}`);
}
