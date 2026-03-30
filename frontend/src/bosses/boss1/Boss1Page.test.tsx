import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, it, expect } from "vitest";
import { Boss1Page } from "./Boss1Page";

describe("Boss1Page", () => {
	it("renders the boss title", () => {
		render(
			<MemoryRouter>
				<Boss1Page />
			</MemoryRouter>,
		);
		expect(screen.getByText("Boss 1: PKCE Missing Attack")).toBeInTheDocument();
	});

	it("renders the two authorization buttons", () => {
		render(
			<MemoryRouter>
				<Boss1Page />
			</MemoryRouter>,
		);
		expect(
			screen.getByText("Authorize WITHOUT PKCE (Vulnerable)"),
		).toBeInTheDocument();
		expect(
			screen.getByText("Authorize WITH PKCE (Protected)"),
		).toBeInTheDocument();
	});

	it("mentions RFC 7636", () => {
		render(
			<MemoryRouter>
				<Boss1Page />
			</MemoryRouter>,
		);
		expect(screen.getByText(/RFC 7636/)).toBeInTheDocument();
	});
});
