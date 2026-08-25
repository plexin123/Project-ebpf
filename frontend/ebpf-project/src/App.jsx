import { useState } from 'react'
import viteLogo from './assets/vite.svg'
import './App.css'

function App() {
  const [count, setCount] = useState(0)

  return (
    <div className="App">
      <header className="App-header">
        <img src={viteLogo} className="App-logo" alt="logo" />
        <p>Hello Vite!</p>
        <button onClick={() => setCount(count + 1)}>
          Count is {count}
        </button>
        <button onClick={() => setCount(sleep + 1)}>
          Sleep is {sleep}
          </button>
      </header>
    </div>
  )
}

export default App
