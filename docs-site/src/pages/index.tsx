import type {ReactNode} from 'react';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Layout from '@theme/Layout';
import styles from './index.module.css';

/* ─────────────────────────────────────────────────────────────────────────────
   Code window — syntax-highlighted terminal demo
   ───────────────────────────────────────────────────────────────────────────── */
function CodeWindow() {
  return (
    <div className={styles.codeWindow}>
      <div className={styles.codeWindowBar}>
        <span className={`${styles.dot} ${styles.dotRed}`} />
        <span className={`${styles.dot} ${styles.dotYellow}`} />
        <span className={`${styles.dot} ${styles.dotGreen}`} />
        <span className={styles.codeWindowTitle}>handlers.yaml</span>
      </div>
      <div className={styles.codeBody}>
        {/* Line 1 */}
        <div>
          <span className={styles.lineNum}>1</span>
          <span className={styles.kw}>handlers</span>
          <span className={styles.pun}>:</span>
        </div>
        {/* Line 2 */}
        <div>
          <span className={styles.lineNum}>2</span>
          {'  '}<span className={styles.fl}>- name</span>
          <span className={styles.pun}>:</span>{' '}
          <span className={styles.str}>deploy-trigger</span>
        </div>
        {/* Line 3 */}
        <div>
          <span className={styles.lineNum}>3</span>
          {'    '}<span className={styles.fl}>match</span>
          <span className={styles.pun}>:</span>
        </div>
        {/* Line 4 */}
        <div>
          <span className={styles.lineNum}>4</span>
          {'      '}<span className={styles.fl}>jq</span>
          <span className={styles.pun}>:</span>{' '}
          <span className={styles.str}>'.type == "deploy"'</span>
        </div>
        {/* Line 5 */}
        <div>
          <span className={styles.lineNum}>5</span>
          {'    '}<span className={styles.fl}>run</span>
          <span className={styles.pun}>:</span>{' '}
          <span className={styles.cmd}>./deploy.sh</span>
        </div>
        {/* Line 6 */}
        <div>
          <span className={styles.lineNum}>6</span>
          {'    '}<span className={styles.fl}>respond</span>
          <span className={styles.pun}>:</span>{' '}
          <span className={styles.str}>'&#123;"ok":&#123;&#123;.ExitCode&#125;&#125;&#125;'</span>
        </div>
        {/* Line 7 blank */}
        <div><span className={styles.lineNum}>7</span></div>
        {/* Line 8 */}
        <div>
          <span className={styles.lineNum}>8</span>
          {'  '}<span className={styles.fl}>- name</span>
          <span className={styles.pun}>:</span>{' '}
          <span className={styles.str}>healthcheck</span>
        </div>
        {/* Line 9 */}
        <div>
          <span className={styles.lineNum}>9</span>
          {'    '}<span className={styles.fl}>match</span>
          <span className={styles.pun}>:</span>{' '}
          <span className={styles.str}>"ping"</span>
        </div>
        {/* Line 10 */}
        <div>
          <span className={styles.lineNum}>10</span>
          {'    '}<span className={styles.fl}>builtin</span>
          <span className={styles.pun}>:</span>{' '}
          <span className={styles.cmd}>echo</span>
        </div>
        {/* Line 11 */}
        <div>
          <span className={styles.lineNum}>11</span>
          {'    '}<span className={styles.fl}>respond</span>
          <span className={styles.pun}>:</span>{' '}
          <span className={styles.str}>'&#123;"pong":"&#123;&#123;now&#125;&#125;"&#125;'</span>
        </div>
        {/* Line 12 blank */}
        <div><span className={styles.lineNum}>12</span></div>
        {/* Line 13 comment */}
        <div>
          <span className={styles.lineNum}>13</span>
          {'  '}<span className={styles.cmt}># 70+ builtins · jq · Lua · pub/sub · KV</span>
        </div>
      </div>
    </div>
  );
}

/* ─────────────────────────────────────────────────────────────────────────────
   Hero
   ───────────────────────────────────────────────────────────────────────────── */
function Hero() {
  return (
    <>
      <div className={styles.heroWrap}>
        <div className={styles.heroBg} />
        <div className={styles.heroInner}>
          {/* Left */}
          <div className={styles.heroLeft}>
            <span className={styles.heroBadge}>
              <span className={styles.heroDot} />
              Open Source · MIT License
            </span>

            <h1 className={styles.heroTitle}>
              WebSocket<br />
              <span className={styles.heroTitleAccent}>Swiss Army Knife</span>
            </h1>

            <p className={styles.heroSub}>
              Bind WebSocket messages to shell pipelines, Go templates, and
              built-in actions — with a full interactive REPL. Think{' '}
              <code className={styles.heroSubCode}>curl</code> +{' '}
              <code className={styles.heroSubCode}>netcat</code> +{' '}
              <code className={styles.heroSubCode}>jq</code> for WebSockets.
            </p>

            <div className={styles.heroActions}>
              <Link to="/docs/" className={styles.btnPrimary}>
                Get started →
              </Link>
              <Link to="/docs/reference/cli" className={styles.btnSecondary}>
                CLI reference
              </Link>
            </div>

            <div className={styles.heroInstall}>
              <span className={styles.heroInstallPrefix}>$</span>
              <span className={styles.heroInstallCmd}>
                go install github.com/0funct0ry/xwebs@latest
              </span>
            </div>
          </div>

          {/* Right: code window */}
          <div className={styles.heroCode}>
            <CodeWindow />
          </div>
        </div>
      </div>

      {/* Wave divider */}
      <div className={styles.heroDivider}>
        <svg viewBox="0 0 1440 48" xmlns="http://www.w3.org/2000/svg" preserveAspectRatio="none">
          <path
            d="M0,48 L0,16 C240,48 480,0 720,16 C960,32 1200,8 1440,16 L1440,48 Z"
            fill="#0A2540"
          />
        </svg>
      </div>
    </>
  );
}

/* ─────────────────────────────────────────────────────────────────────────────
   Stats bar
   ───────────────────────────────────────────────────────────────────────────── */
function StatsBar() {
  const stats = [
    {num: '8',    label: 'Core modes'},
    {num: '70+',  label: 'Built-in actions'},
    {num: '100+', label: 'Template functions'},
    {num: '9',    label: 'Match strategies'},
    {num: '0',    label: 'Dependencies to wire'},
  ];
  return (
    <div className={styles.statsBar}>
      <div className={styles.statsInner}>
        {stats.map(s => (
          <div key={s.label} className={styles.statItem}>
            <div className={styles.statNum}>{s.num}</div>
            <div className={styles.statLabel}>{s.label}</div>
          </div>
        ))}
      </div>
    </div>
  );
}

/* ─────────────────────────────────────────────────────────────────────────────
   Feature cards
   ───────────────────────────────────────────────────────────────────────────── */
const features = [
  {
    icon: '⚡',
    title: 'Shell-first handlers',
    desc: 'Every WebSocket message is an event that triggers your existing toolchain — jq, psql, curl, ffmpeg. Wire them with a YAML file, no SDK needed.',
  },
  {
    icon: '🔧',
    title: 'Go templates everywhere',
    desc: 'Inject message content, env vars, UUIDs, timestamps, HMAC signatures, and KV store values into any field — flags, handlers, or prompts.',
  },
  {
    icon: '💻',
    title: 'Interactive REPL',
    desc: 'Full readline REPL with tab completion, persistent history, syntax highlighting, live handler management, and prompt customization.',
  },
  {
    icon: '📡',
    title: 'Pub/Sub & KV store',
    desc: 'Built-in topic fan-out, sticky broadcasts, per-connection rate limiting, and a server-scoped key-value store — all in-process.',
  },
  {
    icon: '🤖',
    title: '70+ built-in actions',
    desc: 'echo, broadcast, forward, rate-limit, lua scripting, HTTP webhooks, Redis, Kafka, NATS, Ollama/OpenAI, SQLite, S3 — and counting.',
  },
  {
    icon: '🧪',
    title: 'Testing built-in',
    desc: 'Record & replay sessions, load test with bench, diff two server endpoints, run scripted assertions — all without a test framework.',
  },
];

function Features() {
  return (
    <section className={styles.featuresSection}>
      <p className={styles.sectionLabel}>What makes xwebs different</p>
      <h2 className={styles.sectionTitle}>Everything WebSocket tooling should be</h2>
      <p className={styles.sectionSub}>
        Connect once, automate everything. From a one-liner REPL session to a
        production-grade message pipeline.
      </p>
      <div className={styles.featureGrid}>
        {features.map(f => (
          <div key={f.title} className={styles.featureCard}>
            <div className={styles.featureIcon}>{f.icon}</div>
            <div className={styles.featureCardTitle}>{f.title}</div>
            <div className={styles.featureCardDesc}>{f.desc}</div>
          </div>
        ))}
      </div>
    </section>
  );
}

/* ─────────────────────────────────────────────────────────────────────────────
   Quick-start steps
   ───────────────────────────────────────────────────────────────────────────── */
function QuickStart() {
  const steps = [
    {
      label: 'Install',
      code: 'go install github.com/0funct0ry/xwebs@latest',
    },
    {
      label: 'Connect to any WebSocket server',
      code: 'xwebs connect wss://echo.websocket.org',
    },
    {
      label: 'Serve with an inline handler — no config file',
      code: "xwebs serve --port 8080 \\\n  --on '.type == \"ping\" :: respond:{\"type\":\"pong\"}'",
    },
    {
      label: 'Or a full handler config for complex pipelines',
      code: 'xwebs serve --port 8080 --handlers handlers.yaml --interactive',
    },
  ];

  return (
    <section className={styles.quickSection}>
      <div className={styles.quickInner}>
        <h2 className={styles.quickTitle}>Up in 30 seconds</h2>
        <div className={styles.stepList}>
          {steps.map((s, i) => (
            <div key={i} className={styles.step}>
              <div className={styles.stepNum}>{i + 1}</div>
              <div className={styles.stepContent}>
                <div className={styles.stepLabel}>{s.label}</div>
                <pre className={styles.stepCode}>{s.code}</pre>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

/* ─────────────────────────────────────────────────────────────────────────────
   Page root
   ───────────────────────────────────────────────────────────────────────────── */
export default function Home(): ReactNode {
  const {siteConfig} = useDocusaurusContext();
  return (
    <Layout
      title={siteConfig.title}
      description={siteConfig.tagline}>
      <Hero />
      <StatsBar />
      <Features />
      <QuickStart />
    </Layout>
  );
}
