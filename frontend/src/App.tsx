import { BrowserRouter, Routes, Route } from "react-router-dom";
import { BossSelect } from "./components/BossSelect";
import { Boss1Page } from "./bosses/boss1/Boss1Page";

export function App() {
	return (
		<BrowserRouter>
			<Routes>
				<Route path="/" element={<BossSelect />} />
				<Route path="/boss/1" element={<Boss1Page />} />
			</Routes>
		</BrowserRouter>
	);
}
