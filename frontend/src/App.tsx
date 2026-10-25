import { calculatorApi, type CalculatorApi } from './api/client';
import { Calculator } from './components/Calculator';

interface AppProps {
  api?: CalculatorApi;
}

export default function App({ api = calculatorApi }: AppProps) {
  return (
    <main className="app">
      <header className="app-header">
        <h1>Calculator</h1>
        <p>Standard</p>
        <a className="app-docs-link" href="/docs/">
          API docs
        </a>
      </header>
      <Calculator calculate={api.calculate} />
    </main>
  );
}
