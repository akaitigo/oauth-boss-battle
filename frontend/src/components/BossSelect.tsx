import { Link } from "react-router-dom";
import { useBossProgress } from "../hooks/useBossProgress";
import type { BossInfo } from "../types/boss";

const BOSSES: BossInfo[] = [
	{
		id: 1,
		name: "PKCE Missing Attack",
		description:
			"Authorization Code Interception — defeat by implementing PKCE (RFC 7636)",
		defeated: false,
	},
	{
		id: 2,
		name: "State Mismatch (CSRF)",
		description:
			"Cross-Site Request Forgery via missing state parameter (RFC 6749 §10.12)",
		defeated: false,
	},
	{
		id: 3,
		name: "Nonce Replay Attack",
		description: "ID Token replay via nonce reuse (OpenID Connect Core §11.5)",
		defeated: false,
	},
	{
		id: 4,
		name: "JWKS Rotation Failure",
		description:
			"Token verification failure from key rotation issues (RFC 7517)",
		defeated: false,
	},
	{
		id: 5,
		name: "Logout Hell",
		description:
			"Session persistence from incomplete logout implementation (OIDC Session Management)",
		defeated: false,
	},
];

export function BossSelect() {
	const { isDefeated, resetProgress } = useBossProgress();

	const defeatedCount = BOSSES.filter((b) => isDefeated(b.id)).length;

	return (
		<div style={{ maxWidth: 800, margin: "0 auto", padding: 20 }}>
			<h1>OAuth Boss Battle</h1>
			<p>
				Learn OAuth/OIDC security by defeating vulnerability bosses. Each boss
				represents a real-world attack scenario.
			</p>
			<p>
				Progress: {defeatedCount}/{BOSSES.length} bosses defeated
			</p>

			<div style={{ display: "grid", gap: 16, marginTop: 24 }}>
				{BOSSES.map((boss) => {
					const defeated = isDefeated(boss.id);
					const available = boss.id <= 2; // Bosses 1-2 are available
					return (
						<div
							key={boss.id}
							style={{
								border: `2px solid ${defeated ? "#28a745" : available ? "#007bff" : "#6c757d"}`,
								borderRadius: 12,
								padding: 20,
								opacity: available ? 1 : 0.5,
							}}
						>
							<div
								style={{
									display: "flex",
									justifyContent: "space-between",
									alignItems: "center",
								}}
							>
								<h2 style={{ margin: 0 }}>
									Boss {boss.id}: {boss.name}
									{defeated && " ✓"}
								</h2>
								{available && (
									<Link
										to={`/boss/${boss.id}`}
										style={{
											padding: "8px 20px",
											background: defeated ? "#28a745" : "#007bff",
											color: "white",
											textDecoration: "none",
											borderRadius: 8,
										}}
									>
										{defeated ? "Replay" : "Challenge"}
									</Link>
								)}
							</div>
							<p style={{ color: "#666", marginTop: 8 }}>{boss.description}</p>
						</div>
					);
				})}
			</div>

			{defeatedCount > 0 && (
				<button
					onClick={resetProgress}
					style={{
						marginTop: 24,
						padding: "8px 16px",
						background: "#dc3545",
						color: "white",
						border: "none",
						borderRadius: 8,
						cursor: "pointer",
					}}
				>
					Reset Progress
				</button>
			)}
		</div>
	);
}
