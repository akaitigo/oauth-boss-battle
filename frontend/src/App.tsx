import { BrowserRouter, Routes, Route } from "react-router-dom";
import { BossSelect } from "./components/BossSelect";
import { Boss1Page } from "./bosses/boss1/Boss1Page";
import { Boss2Page } from "./bosses/boss2/Boss2Page";

export function App() {
	return (
		<BrowserRouter>
			<Routes>
				<Route path="/" element={<BossSelect />} />
				<Route path="/boss/1" element={<Boss1Page />} />
				<Route path="/boss/2" element={<Boss2Page />} />
			</Routes>
		</BrowserRouter>
	);
}
