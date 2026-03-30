import { render, screen } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { describe, it, expect } from "vitest";
import { Boss4Page } from "./Boss4Page";

describe("Boss4Page", () => {
	it("renders the boss title", () => {
		render(
			<MemoryRouter>
				<Boss4Page />
			</MemoryRouter>,
		);
		expect(
			screen.getByText("Boss 4: JWKS Rotation Failure"),
		).toBeInTheDocument();
	});

	it("renders cache configuration buttons", () => {
		render(
			<MemoryRouter>
				<Boss4Page />
			</MemoryRouter>,
		);
		expect(screen.getByText("Stale Cache (Vulnerable)")).toBeInTheDocument();
		expect(screen.getByText("Smart Cache (Protected)")).toBeInTheDocument();
	});

	it("mentions RFC 7517", () => {
		render(
			<MemoryRouter>
				<Boss4Page />
			</MemoryRouter>,
		);
		expect(screen.getByText(/RFC 7517/)).toBeInTheDocument();
	});
});
