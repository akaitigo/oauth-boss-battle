import { useState } from "react";
import { useBossProgress } from "../../hooks/useBossProgress";
import type { BossResult } from "../../types/boss";

const API_BASE = "/api/boss/5";

type Phase = "intro" | "loggedIn" | "logout" | "result";

interface SessionInfo {
	sid: string;
	rp_id: string;
	rp_name: string;
	user_id: string;
	active: boolean;
	logout_by: string;
}

export function Boss5Page() {
	const { isDefeated, markDefeated } = useBossProgress();
	const [phase, setPhase] = useState<Phase>("intro");
	const [sidPrefix, setSidPrefix] = useState("");
	const [sessions, setSessions] = useState<SessionInfo[]>([]);
	const [result, setResult] = useState<BossResult | null>(null);
	const [usedBackChannel, setUsedBackChannel] = useState(false);
	const [logoutMessage, setLogoutMessage] = useState("");

	const defeated = isDefeated(5);

	const handleLogin = async () => {
		const res = await fetch(`${API_BASE}/login`, {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({ user_id: "demo-user" }),
		});
		const data: Record<string, unknown> = await res.json();
		setSidPrefix(data.sid_prefix as string);
		setSessions(data.sessions as SessionInfo[]);
		setUsedBackChannel(false);
		setLogoutMessage("");
		setPhase("loggedIn");
	};

	const handleFrontChannelLogout = async () => {
		const res = await fetch(`${API_BASE}/logout-frontchannel`, {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({ sid_prefix: sidPrefix }),
		});
		const data: Record<string, unknown> = await res.json();
		setSessions(data.sessions as SessionInfo[]);
		setLogoutMessage(data.message as string);
		setPhase("logout");
	};

	const handleBackChannelLogout = async () => {
		const res = await fetch(`${API_BASE}/logout-backchannel`, {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({ sid_prefix: sidPrefix }),
		});
		const data: Record<string, unknown> = await res.json();
		setSessions(data.sessions as SessionInfo[]);
		setLogoutMessage(data.message as string);
		setUsedBackChannel(true);
		const isDefeatedNow = data.defeated as boolean;
		if (isDefeatedNow) markDefeated(5);
		setPhase("logout");
	};

	const handleVerify = async () => {
		const res = await fetch(`${API_BASE}/verify`, {
			method: "POST",
			headers: { "Content-Type": "application/json" },
			body: JSON.stringify({
				sid_prefix: sidPrefix,
				used_back_channel: usedBackChannel,
			}),
		});
		const data: BossResult = await res.json();
		setResult(data);
		if (data.defeated) markDefeated(5);
		setPhase("result");
	};

	const handleReset = () => {
		setPhase("intro");
		setSidPrefix("");
		setSessions([]);
		setResult(null);
		setUsedBackChannel(false);
		setLogoutMessage("");
	};

	const renderSessions = () => (
		<div style={{ display: "grid", gap: 12, marginBottom: 16 }}>
			{sessions.map((s) => (
				<div
					key={s.sid}
					style={{
						border: `2px solid ${s.active ? "#dc3545" : "#28a745"}`,
						borderRadius: 8,
						padding: 12,
						display: "flex",
						justifyContent: "space-between",
						alignItems: "center",
					}}
				>
					<div>
						<strong>{s.rp_name}</strong> ({s.rp_id})
						<br />
						<small>SID: {s.sid}</small>
					</div>
					<div
						style={{
							padding: "4px 12px",
							borderRadius: 4,
							background: s.active ? "#dc3545" : "#28a745",
							color: "white",
							fontWeight: "bold",
						}}
					>
						{s.active ? "ACTIVE" : `Logged out (${s.logout_by})`}
					</div>
				</div>
			))}
		</div>
	);

	return (
		<div style={{ maxWidth: 800, margin: "0 auto", padding: 20 }}>
			<h1>Boss 5: Logout Hell</h1>
			<p>
				Incomplete logout leaves sessions active at Relying Parties — OpenID
				Connect Back-Channel Logout ensures reliable session termination.
			</p>

			{defeated && (
				<div
					style={{
						background: "#d4edda",
						border: "1px solid #c3e6cb",
						padding: 16,
						borderRadius: 8,
						marginBottom: 16,
					}}
				>
					Boss Defeated!
				</div>
			)}

			{phase === "intro" && (
				<div>
					<h2>Step 1: Login to Multiple RPs</h2>
					<p>
						Simulate SSO login that creates sessions at 3 Relying Parties
						(Email, Calendar, Files).
					</p>
					<button
						onClick={handleLogin}
						style={{
							padding: "12px 24px",
							background: "#007bff",
							color: "white",
							border: "none",
							borderRadius: 8,
							cursor: "pointer",
						}}
					>
						Login (SSO)
					</button>
				</div>
			)}

			{phase === "loggedIn" && (
				<div>
					<h2>Step 2: RP Sessions</h2>
					{renderSessions()}
					<h2>Step 3: Choose Logout Method</h2>
					<div style={{ display: "flex", gap: 16 }}>
						<button
							onClick={handleFrontChannelLogout}
							style={{
								padding: "12px 24px",
								background: "#dc3545",
								color: "white",
								border: "none",
								borderRadius: 8,
								cursor: "pointer",
							}}
						>
							Front-Channel Logout (Unreliable)
						</button>
						<button
							onClick={handleBackChannelLogout}
							style={{
								padding: "12px 24px",
								background: "#28a745",
								color: "white",
								border: "none",
								borderRadius: 8,
								cursor: "pointer",
							}}
						>
							Back-Channel Logout (Reliable)
						</button>
					</div>
				</div>
			)}

			{phase === "logout" && (
				<div>
					<h2>Logout Result</h2>
					{logoutMessage && (
						<div
							style={{
								background: "#f8f9fa",
								padding: 16,
								borderRadius: 8,
								marginBottom: 16,
							}}
						>
							<p>{logoutMessage}</p>
						</div>
					)}
					{renderSessions()}
					<div style={{ display: "flex", gap: 16 }}>
						<button
							onClick={handleVerify}
							style={{
								padding: "12px 24px",
								background: "#007bff",
								color: "white",
								border: "none",
								borderRadius: 8,
								cursor: "pointer",
							}}
						>
							Check Boss Status
						</button>
						<button
							onClick={handleReset}
							style={{
								padding: "12px 24px",
								background: "#6c757d",
								color: "white",
								border: "none",
								borderRadius: 8,
								cursor: "pointer",
							}}
						>
							Try Again
						</button>
					</div>
				</div>
			)}

			{phase === "result" && result && (
				<div>
					<h2>Result</h2>
					<div
						style={{
							background: result.defeated ? "#d4edda" : "#f8d7da",
							border: `1px solid ${result.defeated ? "#c3e6cb" : "#f5c6cb"}`,
							padding: 16,
							borderRadius: 8,
							marginBottom: 16,
						}}
					>
						<strong>
							{result.defeated ? "BOSS DEFEATED!" : "Sessions Still Active..."}
						</strong>
						<p>{result.message}</p>
						{result.explanation && (
							<p style={{ color: "#666", whiteSpace: "pre-line" }}>
								{result.explanation}
							</p>
						)}
						{result.rfc_link && (
							<p>
								<a
									href={result.rfc_link}
									target="_blank"
									rel="noopener noreferrer"
								>
									OpenID Connect Back-Channel Logout
								</a>
							</p>
						)}
					</div>
					<button
						onClick={handleReset}
						style={{
							padding: "12px 24px",
							background: "#007bff",
							color: "white",
							border: "none",
							borderRadius: 8,
							cursor: "pointer",
						}}
					>
						Try Again
					</button>
				</div>
			)}
		</div>
	);
}
