import './App.css';
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import WeatherStatistics from './weatherStatistics/WeatherStatistics';

function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route path="/" element={<WeatherStatistics/>}/>
      </Routes>
    </BrowserRouter>
  );
}

export default App;
