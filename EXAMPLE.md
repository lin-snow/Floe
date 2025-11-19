# 这里存放着对每一个 example.md 的解释

## 01_basic_messaging.yaml

**Purpose**: Demonstrates basic linear execution and message passing between steps.
**Scenario**:

1.  `step_1_echo`: Takes an input string, processes it (mock summary), and outputs a message `info: "Step 1 completed"`.
2.  `step_2_process`: Reads the output from Step 1 and the message from Step 1 (`${messages.info}`).
    **Expected Result**:

- Step 1 runs first.
- Step 2 runs second, successfully interpolating the message from Step 1.
- Trace confirms the sequence and data flow.

## 02_parallel_aggregation.yaml

**Purpose**: Demonstrates parallel execution and result aggregation.
**Scenario**:

1.  `parallel_process`: Runs two branches (`branch_a` and `branch_b`) concurrently. Each branch processes the input data.
2.  `aggregate`: Runs after the parallel step completes, reading outputs from both branches (`${global.res_a}`, `${global.res_b}`) and combining them.
    **Expected Result**:

- `branch_a` and `branch_b` run in the same superstep (conceptually, though current runtime might show them as one "parallel" step execution).
- `aggregate` runs in the next superstep.
- Final output contains data from both branches.

## 03_flow_control_next.yaml

**Purpose**: Demonstrates flow control using the `next` field to skip steps.
**Scenario**:

1.  `step_start`: Executes and explicitly points to `step_end` using `next: step_end`.
2.  `step_middle`: This step is defined in the sequence but should be skipped because `step_start` jumped over it.
3.  `step_end`: Executes after `step_start`.
    **Expected Result**:

- `step_start` runs.
- `step_middle` is NOT executed.
- `step_end` runs.
- Trace shows only start and end steps.
