import { BrowserRouter, Routes, Route } from "react-router-dom";
import { Layout } from "./components/Layout";
import Overview from "./pages/Overview";
import Services from "./pages/Services";
import Incidents from "./pages/Incidents";
import Alerts from "./pages/Alerts";
import Traces from "./pages/Traces";
import Logs from "./pages/Logs";
import Capacity from "./pages/Capacity";
import Runbooks from "./pages/Runbooks";

export default function App() {
  return (
    <BrowserRouter>
      <Routes>
        <Route element={<Layout />}>
          <Route path="/" element={<Overview />} />
          <Route path="/services" element={<Services />} />
          <Route path="/incidents" element={<Incidents />} />
          <Route path="/alerts" element={<Alerts />} />
          <Route path="/traces" element={<Traces />} />
          <Route path="/logs" element={<Logs />} />
          <Route path="/capacity" element={<Capacity />} />
          <Route path="/runbooks" element={<Runbooks />} />
        </Route>
      </Routes>
    </BrowserRouter>
  );
}
