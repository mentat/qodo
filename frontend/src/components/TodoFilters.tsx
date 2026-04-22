import { Group, Select, SegmentedControl } from '@mantine/core';

interface TodoFiltersProps {
  priority: string | null;
  category: string | null;
  status: string;
  categories: string[];
  onPriorityChange: (value: string | null) => void;
  onCategoryChange: (value: string | null) => void;
  onStatusChange: (value: string) => void;
}

export function TodoFilters({
  priority,
  category,
  status,
  categories,
  onPriorityChange,
  onCategoryChange,
  onStatusChange,
}: TodoFiltersProps) {
  return (
    <Group mb="md" wrap="wrap">
      <Select
        placeholder="Priority"
        clearable
        value={priority}
        onChange={onPriorityChange}
        data={[
          { value: 'high', label: 'High' },
          { value: 'medium', label: 'Medium' },
          { value: 'low', label: 'Low' },
        ]}
        w={140}
      />

      <Select
        placeholder="Category"
        clearable
        value={category}
        onChange={onCategoryChange}
        data={categories.map((c) => ({ value: c, label: c }))}
        w={160}
      />

      <SegmentedControl
        value={status}
        onChange={onStatusChange}
        data={[
          { label: 'All', value: 'all' },
          { label: 'Active', value: 'active' },
          { label: 'Done', value: 'done' },
        ]}
      />
    </Group>
  );
}
