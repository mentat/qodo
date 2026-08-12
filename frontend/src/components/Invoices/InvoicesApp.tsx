import { useEffect, useMemo } from 'react';
import {
  Box,
  Badge,
  Button,
  Group,
  Paper,
  ScrollArea,
  Select,
  Stack,
  Text,
  Center,
  Loader,
  Tooltip,
} from '@mantine/core';
import { IconFileInvoice, IconCheck, IconX, IconCreditCard } from '@tabler/icons-react';
import { notifications } from '@mantine/notifications';
import dayjs from 'dayjs';
import { useInvoiceStore, awaitingApprovalTotal } from '../../store/invoiceStore';
import { formatCents, type Invoice, type InvoiceStatus } from '../../types/invoice';

const STATUS_COLORS: Record<InvoiceStatus, string> = {
  draft: 'gray',
  pending_approval: 'yellow',
  approved: 'neonGreen',
  scheduled: 'synthPurple',
  paid: 'neonGreen',
  rejected: 'red',
};

const STATUS_OPTIONS = [
  { value: 'all', label: 'All statuses' },
  { value: 'pending_approval', label: 'Awaiting approval' },
  { value: 'approved', label: 'Approved' },
  { value: 'scheduled', label: 'Scheduled' },
  { value: 'paid', label: 'Paid' },
  { value: 'rejected', label: 'Rejected' },
];

// Entities the demo user has visibility into. Real deployments read this from
// the org's entity list.
const ENTITY_OPTIONS = [
  { value: 'entity-us-1', label: 'Northwind US' },
  { value: 'entity-us-2', label: 'Northwind West' },
  { value: 'entity-eu-1', label: 'Northwind EU' },
];

function nextApprover(inv: Invoice): string | null {
  const step = inv.approvalChain.find((s) => s.decision === 'pending');
  return step ? step.role : null;
}

export function InvoicesApp() {
  const invoices = useInvoiceStore((s) => s.invoices);
  const policy = useInvoiceStore((s) => s.policy);
  const entityId = useInvoiceStore((s) => s.entityId);
  const status = useInvoiceStore((s) => s.status);
  const loading = useInvoiceStore((s) => s.loading);
  const pending = useInvoiceStore((s) => s.pending);
  const fetch = useInvoiceStore((s) => s.fetch);
  const fetchPolicy = useInvoiceStore((s) => s.fetchPolicy);
  const setEntity = useInvoiceStore((s) => s.setEntity);
  const setStatus = useInvoiceStore((s) => s.setStatus);
  const approve = useInvoiceStore((s) => s.approve);
  const reject = useInvoiceStore((s) => s.reject);
  const pay = useInvoiceStore((s) => s.pay);

  useEffect(() => {
    void fetchPolicy();
  }, [fetchPolicy]);

  // Refetch whenever the entity or status filter changes — the server applies
  // both, so we never filter client-side.
  useEffect(() => {
    void fetch();
  }, [fetch, entityId, status]);

  const totalAwaiting = useMemo(() => awaitingApprovalTotal(invoices), [invoices]);

  const act = async (label: string, fn: () => Promise<void>) => {
    try {
      await fn();
      notifications.show({ title: label, message: 'Invoice updated', color: 'neonGreen' });
    } catch (e) {
      notifications.show({ title: 'Error', message: (e as Error).message, color: 'red' });
    }
  };

  return (
    <Stack gap="md" style={{ height: 'calc(100vh - 92px)' }}>
      <Group justify="space-between" align="flex-end">
        <Group gap="sm">
          <Select
            label="Entity"
            data={ENTITY_OPTIONS}
            value={entityId}
            onChange={(v) => v && setEntity(v)}
            allowDeselect={false}
            w={180}
          />
          <Select
            label="Status"
            data={STATUS_OPTIONS}
            value={status}
            onChange={(v) => v && setStatus(v as InvoiceStatus | 'all')}
            allowDeselect={false}
            w={180}
          />
        </Group>
        <Stack gap={2} align="flex-end">
          <Text size="sm" c="dimmed">
            Awaiting approval
          </Text>
          <Text size="xl" fw={700}>
            {formatCents(totalAwaiting)}
          </Text>
          {policy && (
            <Text size="11px" c="dimmed">
              Self-approval up to {formatCents(policy.delegatedLimitCents)}
            </Text>
          )}
        </Stack>
      </Group>

      {loading && invoices.length === 0 ? (
        <Center style={{ flex: 1 }}>
          <Loader color="synthPurple" />
        </Center>
      ) : invoices.length === 0 ? (
        <Center style={{ flex: 1 }}>
          <Stack align="center" gap="xs">
            <IconFileInvoice size={40} stroke={1.4} opacity={0.4} />
            <Text c="dimmed">No invoices for this entity</Text>
          </Stack>
        </Center>
      ) : (
        <ScrollArea style={{ flex: 1 }} type="auto">
          <Stack gap="xs">
            {invoices.map((inv) => {
              const busy = pending.includes(inv.id);
              const role = nextApprover(inv);
              return (
                <Paper key={inv.id} p="md" radius="md" withBorder>
                  <Group justify="space-between" wrap="nowrap">
                    <Box style={{ minWidth: 0 }}>
                      <Group gap="xs">
                        <Text fw={600} truncate>
                          {inv.vendorName}
                        </Text>
                        <Badge size="sm" color={STATUS_COLORS[inv.status]} variant="light">
                          {inv.status.replace('_', ' ')}
                        </Badge>
                        {role && (
                          <Badge size="sm" variant="outline" color="gray">
                            next: {role}
                          </Badge>
                        )}
                      </Group>
                      <Text size="12px" c="dimmed">
                        {inv.invoiceNumber || 'no number'} · {inv.lines.length} line
                        {inv.lines.length === 1 ? '' : 's'}
                        {inv.dueDate ? ` · due ${dayjs(inv.dueDate).format('MMM D')}` : ''}
                        {inv.paymentRef ? ` · ${inv.paymentRef}` : ''}
                      </Text>
                    </Box>

                    <Group gap="sm" wrap="nowrap">
                      <Text fw={700} style={{ whiteSpace: 'nowrap' }}>
                        {formatCents(inv.amountCents, inv.currencyCode)}
                      </Text>
                      {inv.status === 'pending_approval' && (
                        <>
                          <Tooltip label="Approve">
                            <Button
                              size="xs"
                              variant="light"
                              color="neonGreen"
                              loading={busy}
                              disabled={busy}
                              leftSection={<IconCheck size={14} />}
                              onClick={() => void act('Approved', () => approve(inv.id))}
                            >
                              Approve
                            </Button>
                          </Tooltip>
                          <Tooltip label="Reject">
                            <Button
                              size="xs"
                              variant="subtle"
                              color="red"
                              loading={busy}
                              disabled={busy}
                              leftSection={<IconX size={14} />}
                              onClick={() => void act('Rejected', () => reject(inv.id))}
                            >
                              Reject
                            </Button>
                          </Tooltip>
                        </>
                      )}
                      {inv.status === 'approved' && (
                        <Button
                          size="xs"
                          variant="light"
                          color="synthPurple"
                          loading={busy}
                          disabled={busy}
                          leftSection={<IconCreditCard size={14} />}
                          onClick={() => void act('Submitted', () => pay(inv.id))}
                        >
                          Submit payment
                        </Button>
                      )}
                    </Group>
                  </Group>
                </Paper>
              );
            })}
          </Stack>
        </ScrollArea>
      )}
    </Stack>
  );
}
