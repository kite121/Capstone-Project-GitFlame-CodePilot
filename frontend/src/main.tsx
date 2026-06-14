import React from "react";
import { createRoot } from "react-dom/client";
import { Bot, FileSearch, GitPullRequest } from "lucide-react";
import "./styles.css";

function App() {
  return (
    <main className="page">
      <section className="header">
        <div>
          <p className="eyebrow">GitFlame CodePilot</p>
          <h1>Repository AI workflow demo</h1>
        </div>
        <button className="primary-button">
          <Bot size={18} />
          Work with AI
        </button>
      </section>

      <section className="grid">
        <article className="panel">
          <FileSearch size={22} />
          <h2>.yml analysis</h2>
          <p>Detect configuration state and validate repository AI settings.</p>
        </article>
        <article className="panel">
          <Bot size={22} />
          <h2>Issue plan</h2>
          <p>Generate a Markdown implementation plan from issue context.</p>
        </article>
        <article className="panel">
          <GitPullRequest size={22} />
          <h2>PR payload</h2>
          <p>Prepare generated files, branch payload, and reviewer assignment.</p>
        </article>
      </section>
    </main>
  );
}

createRoot(document.getElementById("root")!).render(<App />);

