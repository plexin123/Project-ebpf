
import {GraphProvider} from './state/GraphContext'
import { Dashboard } from './components/Dashboard'
import './App.css'

function App() {
  return ( 
    <GraphProvider>
      <Dashboard>
        </Dashboard>
    </GraphProvider> 
  )
}

export default App
