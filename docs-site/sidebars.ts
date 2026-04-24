import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

const sidebars: SidebarsConfig = {
  docs: [
    {
      type: 'category',
      label: 'Getting Started',
      collapsed: false,
      items: ['intro', 'getting-started'],
    },
    {
      type: 'category',
      label: 'Client Mode',
      items: ['client/repl', 'client/non-interactive'],
    },
    {
      type: 'category',
      label: 'Server Mode',
      items: ['server/repl', 'server/pubsub'],
    },
    {
      type: 'category',
      label: 'Handlers',
      items: [
        'handlers/yaml-schema',
        'handlers/inline',
        'handlers/match-strategies',
      ],
    },
    {
      type: 'category',
      label: 'Templates',
      items: [
        'templates/variables',
        'templates/functions',
        'templates/prompt',
      ],
    },
    {
      type: 'category',
      label: 'Builtins',
      items: [
        'builtins/index',
        {
          type: 'category',
          label: 'Core',
          items: [
            'builtins/echo',
            'builtins/broadcast',
            'builtins/broadcast-others',
            'builtins/forward',
            'builtins/sequence',
            'builtins/template',
          ],
        },
        {
          type: 'category',
          label: 'File & State',
          items: [
            'builtins/file-send',
            'builtins/file-write',
            'builtins/kv-set',
            'builtins/kv-get',
            'builtins/kv-del',
            'builtins/kv-list',
          ],
        },
        {
          type: 'category',
          label: 'Control Flow',
          items: [
            'builtins/rate-limit',
            'builtins/delay',
            'builtins/drop',
            'builtins/close',
            'builtins/debounce',
            'builtins/gate',
            'builtins/once',
            'builtins/sample',
            'builtins/rule-engine',
          ],
        },
        {
          type: 'category',
          label: 'Pub/Sub',
          items: [
            'builtins/publish',
            'builtins/subscribe',
            'builtins/unsubscribe',
            'builtins/throttle-broadcast',
            'builtins/multicast',
            'builtins/sticky-broadcast',
            'builtins/round-robin',
          ],
        },
        {
          type: 'category',
          label: 'Routing',
          items: [
            'builtins/ab-test',
            'builtins/shadow',
          ],
        },
        {
          type: 'category',
          label: 'Observability',
          items: [
            'builtins/log',
            'builtins/metric',
          ],
        },
        {
          type: 'category',
          label: 'HTTP & Webhooks',
          items: [
            'builtins/http',
            'builtins/http-get',
            'builtins/http-graphql',
            'builtins/webhook',
            'builtins/webhook-hmac',
          ],
        },
        {
          type: 'category',
          label: 'AI / LLM',
          items: [
            'builtins/ollama-generate',
            'builtins/ollama-chat',
            'builtins/ollama-embed',
            'builtins/ollama-classify',
            'builtins/openai-chat',
          ],
        },
        {
          type: 'category',
          label: 'Messaging',
          items: [
            'builtins/mqtt-publish',
            'builtins/mqtt-subscribe',
            'builtins/nats-publish',
            'builtins/nats-subscribe',
            'builtins/kafka-produce',
            'builtins/kafka-consume',
          ],
        },
        {
          type: 'category',
          label: 'Storage',
          items: [
            'builtins/sqlite',
            'builtins/postgres',
            'builtins/redis-set',
            'builtins/redis-get',
            'builtins/redis-del',
            'builtins/redis-publish',
            'builtins/redis-subscribe',
            'builtins/redis-lpush',
            'builtins/redis-rpop',
            'builtins/redis-incr',
            'builtins/append-file',
            'builtins/s3-put',
            'builtins/s3-get',
          ],
        },
        {
          type: 'category',
          label: 'Transformation',
          items: [
            'builtins/jq-transform',
            'builtins/json-merge',
            'builtins/xml-to-json',
            'builtins/csv-parse',
            'builtins/schema-validate',
          ],
        },
        {
          type: 'category',
          label: 'Scripting',
          items: ['builtins/lua'],
        },
      ],
    },
    {
      type: 'category',
      label: 'Examples',
      items: [
        'examples/index',
        'examples/live-shell-terminal',
        'examples/lua',
      ],
    },
    {
      type: 'category',
      label: 'CLI Reference',
      items: ['reference/cli', 'reference/binary-frames'],
    },
  ],
};

export default sidebars;
