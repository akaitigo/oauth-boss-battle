import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, it, expect } from "vitest";
import { Boss3Page } from "./Boss3Page";

describe("Boss3Page", () => {
	it("renders the boss title", () => {
		render(
			<MemoryRouter>
				<Boss3Page />
			</MemoryRouter>,
		);
		expect(screen.getByText("Boss 3: Nonce Replay Attack")).toBeInTheDocument();
	});

	it("renders authorization buttons", () => {
		render(
			<MemoryRouter>
				<Boss3Page />
			</MemoryRouter>,
		);
		expect(
			screen.getByText("Authorize WITHOUT nonce (Vulnerable)"),
		).toBeInTheDocument();
		expect(
			screen.getByText("Authorize WITH nonce (Protected)"),
		).toBeInTheDocument();
	});

	it("mentions OpenID Connect Core", () => {
		render(
			<MemoryRouter>
				<Boss3Page />
			</MemoryRouter>,
		);
		expect(screen.getByText(/OpenID Connect Core/)).toBeInTheDocument();
	});
});
