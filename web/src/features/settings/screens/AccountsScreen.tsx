// features/settings/screens/AccountsScreen.tsx — Settings → Accounts (plan 2.2). Lists the server's accounts and
// does the four things the server allows: create, disable/enable, reset a password, delete.
//
// Two rules decide the whole shape of this screen, and both come from the server rather than from taste:
//   * it is administrator-only, so a signed-in user who is not an administrator is told so and nothing else — the
//     screen must never call the endpoint to "check" (the list itself is the leak);
//   * an account can only be deleted once it is *disabled*, and the server wants its exact username typed back.
//     So Disable is offered first and Delete disappears until it's disabled — the UI follows the rule instead of
//     letting someone press a button that always fails.
//
// Inline over modal: destructive actions open in place, under the row they belong to, rather than in a dialog on
// top of a list. On a phone that keeps the account being acted on visible while you type its name, which is the
// one thing a confirm dialog usually gets wrong.
import { useState } from "react";
import { ChevronDown, ChevronRight, UserPlus } from "lucide-react";
import { Page, Stack, Inline, Heading, Text, Card, Button, TextField, PillRow, Menu, Async } from "../../../ui";
import { useSession } from "../../auth/useSession";
import { useAccounts, useAccountMutations } from "../useAccounts";
import { ACCOUNT_ROLES, messageFromError, type AccountRole } from "../api/accounts";
import type { Account } from "../../auth/api/session";

type RowMode = { kind: "idle" } | { kind: "reset" } | { kind: "delete" };

export function AccountsScreen() {
  const session = useSession();

  return (
    <Page>
      <Stack gap={24}>
        <Heading level={1}>Accounts</Heading>
        <Async query={session}>{(me) => <AccountsBody isAdministrator={me.role === "administrator"} />}</Async>
      </Stack>
    </Page>
  );
}

/** Split out so the role check happens inside <Async> (law L5: a screen never branches on `isLoading` itself) and
 *  so a non-administrator never mounts the component that would call the administrator-only endpoint. */
function AccountsBody({ isAdministrator }: { isAdministrator: boolean }) {
  const mutations = useAccountMutations();
  const accounts = useAccounts(isAdministrator);

  const [creating, setCreating] = useState(false);
  const [username, setUsername] = useState("");
  const [password, setPassword] = useState("");
  const [role, setRole] = useState<AccountRole>("user");
  const [rowModes, setRowModes] = useState<Record<string, RowMode>>({});

  const currentId = accounts.data?.current_account_id;

  if (!isAdministrator) {
    return (
      <Text muted>
        Only an administrator can see or change accounts on this server.
      </Text>
    );
  }

  function setRowMode(id: string, mode: RowMode) {
    mutations.reset();
    setRowModes((previous) => ({ ...previous, [id]: mode }));
  }

  return (
    <Stack gap={24}>
        <Text muted>
          Everyone who can sign in to this server. Disabling an account ends its sign-ins immediately; deleting
          is permanent and needs the account disabled first.
        </Text>

        {mutations.error !== null && mutations.error !== undefined && (
          <Text size="13" tone="error">{messageFromError(mutations.error)}</Text>
        )}

        {/* Create — an inline form rather than a dialog, so the new account appears in the list it belongs to. */}
        {creating ? (
          <Card>
            <Stack gap={12}>
              <Heading level={2} size="16">New account</Heading>
              <TextField
                label="Username"
                value={username}
                autoComplete="off"
                autoCapitalize="none"
                spellCheck={false}
                onChange={(event) => setUsername(event.target.value)}
              />
              <TextField
                label="Password"
                type="password"
                value={password}
                autoComplete="new-password"
                onChange={(event) => setPassword(event.target.value)}
              />
              <Text size="11" muted>10 to 72 characters.</Text>
              <Stack gap={8}>
                <Text size="13" muted>Role</Text>
                <PillRow items={ACCOUNT_ROLES} activeId={role} onSelect={(id) => setRole(id as AccountRole)} />
              </Stack>
              <Inline gap={8}>
                <Button
                  variant="primary"
                  disabled={username.trim() === "" || password.length < 10 || mutations.isPending}
                  onClick={() =>
                    mutations.create({ username: username.trim(), password, role }, () => {
                      // Only on success: a failed create must leave what was typed on screen so it can be corrected.
                      setCreating(false);
                      setUsername("");
                      setPassword("");
                      setRole("user");
                    })
                  }
                >
                  Create account
                </Button>
                <Button
                  variant="secondary"
                  onClick={() => {
                    setCreating(false);
                    setUsername("");
                    setPassword("");
                    setRole("user");
                    mutations.reset();
                  }}
                >
                  Cancel
                </Button>
              </Inline>
            </Stack>
          </Card>
        ) : (
          <Inline gap={8}>
            <Button variant="secondary" onClick={() => setCreating(true)}>
              <UserPlus size={16} aria-hidden="true" /> Add an account
            </Button>
          </Inline>
        )}

        <Async query={accounts}>
          {(data) => (
            <Stack gap={12}>
              {data.items.map((account) => (
                <AccountRow
                  key={account.id}
                  account={account}
                  isCurrent={account.id === data.current_account_id}
                  mode={rowModes[account.id] ?? { kind: "idle" }}
                  onModeChange={(mode) => setRowMode(account.id, mode)}
                  mutations={mutations}
                />
              ))}
              <Text size="11" muted>
                {currentId === undefined ? "" : "You are signed in as one of these accounts. "}
                At least one enabled administrator must always remain.
              </Text>
            </Stack>
          )}
        </Async>
    </Stack>
  );
}

function AccountRow({
  account,
  isCurrent,
  mode,
  onModeChange,
  mutations,
}: {
  account: Account;
  isCurrent: boolean;
  mode: RowMode;
  onModeChange: (mode: RowMode) => void;
  mutations: ReturnType<typeof useAccountMutations>;
}) {
  const [resetPassword, setResetPassword] = useState("");
  const [confirmation, setConfirmation] = useState("");

  const actions = [
    { id: account.disabled ? "enable" : "disable", label: account.disabled ? "Enable" : "Disable", section: "Access" },
    { id: "reset", label: "Reset password", section: "Access" },
    // Only offered once it is disabled — the server refuses to delete an enabled account, so offering it earlier
    // would be a button that can only fail.
    ...(account.disabled ? [{ id: "delete", label: "Delete permanently", section: "Danger" }] : []),
  ];

  function onAction(id: string) {
    if (id === "enable" || id === "disable") {
      mutations.act(account.id, account.disabled ? "enable" : "disable");
      return;
    }
    if (id === "reset") {
      setResetPassword("");
      onModeChange({ kind: "reset" });
      return;
    }
    if (id === "delete") {
      setConfirmation("");
      onModeChange({ kind: "delete" });
    }
  }

  return (
    <Card>
      <Stack gap={12}>
        <Inline gap={12} align="center">
          <div style={{ flex: 1, minWidth: 0 }}>
            <Stack gap={4}>
              <Text as="span" size="14">
                {account.username}
                {isCurrent ? " (you)" : ""}
              </Text>
              <Text size="11" muted>
                {account.role === "administrator" ? "Administrator" : "User"}
                {account.disabled ? " · Disabled" : " · Active"}
              </Text>
            </Stack>
          </div>
          <Menu
            label={`Actions for ${account.username}`}
            icon={mode.kind === "idle" ? ChevronRight : ChevronDown}
            items={actions}
            onSelect={onAction}
          />
        </Inline>

        {mode.kind === "reset" && (
          <Stack gap={8}>
            <TextField
              label={`New password for ${account.username}`}
              type="password"
              value={resetPassword}
              autoComplete="new-password"
              onChange={(event) => setResetPassword(event.target.value)}
            />
            <Text size="11" muted>10 to 72 characters. Every existing sign-in for this account ends.</Text>
            <Inline gap={8}>
              <Button
                variant="primary"
                disabled={resetPassword.length < 10 || mutations.isPending}
                onClick={() => {
                  mutations.act(account.id, "reset_password", resetPassword);
                  setResetPassword("");
                  onModeChange({ kind: "idle" });
                }}
              >
                Reset password
              </Button>
              <Button variant="secondary" onClick={() => onModeChange({ kind: "idle" })}>Cancel</Button>
            </Inline>
          </Stack>
        )}

        {mode.kind === "delete" && (
          <Stack gap={8}>
            <Text size="13" tone="error">
              This deletes {account.username} and everything it stored on this server. It cannot be undone.
            </Text>
            <TextField
              label={`Type ${account.username} to confirm`}
              value={confirmation}
              autoComplete="off"
              autoCapitalize="none"
              spellCheck={false}
              onChange={(event) => setConfirmation(event.target.value)}
            />
            <Inline gap={8}>
              <Button
                variant="primary"
                disabled={confirmation !== account.username || mutations.isPending}
                onClick={() => {
                  mutations.remove(account.id, confirmation);
                  setConfirmation("");
                  onModeChange({ kind: "idle" });
                }}
              >
                Delete permanently
              </Button>
              <Button variant="secondary" onClick={() => onModeChange({ kind: "idle" })}>Cancel</Button>
            </Inline>
          </Stack>
        )}
      </Stack>
    </Card>
  );
}
