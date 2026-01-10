package mcp

import (
	"context"
	"fmt"

	"github.com/felixgeelhaar/mcp-go"
)

// RegisterPrompts registers MCP prompts for common Orbita workflows.
func RegisterPrompts(srv *mcp.Server, deps ToolDependencies) error {
	if srv == nil {
		return fmt.Errorf("server is required")
	}

	// Daily planning prompt
	srv.Prompt("daily_planning").
		Description("Guide for planning your day with Orbita. Helps prioritize tasks, schedule time blocks, and set daily goals.").
		Handler(func(ctx context.Context, args map[string]string) (*mcp.PromptResult, error) {
			return &mcp.PromptResult{
				Description: "Daily Planning Session",
				Messages: []mcp.PromptMessage{
					{
						Role: string(mcp.RoleUser),
						Content: mcp.TextContent{
							Type: "text",
							Text: `Help me plan my day effectively. Please:

1. First, check my schedule for today using the orbita://schedule/today resource
2. Review my overdue tasks using the orbita://tasks/overdue resource
3. Look at tasks due today using the orbita://tasks/today resource
4. Check my active habits using the orbita://habits/active resource

Based on this information:
- Identify the top 3 priority tasks I should focus on
- Suggest optimal time blocks for deep work
- Highlight any scheduling conflicts
- Recommend which habits to prioritize today

If there are overdue tasks, help me decide:
- Which to reschedule to today
- Which to delegate or defer
- Which might need to be archived

Please provide actionable recommendations I can implement using the task.* and schedule.* tools.`,
						},
					},
				},
			}, nil
		})

	// Weekly review prompt
	srv.Prompt("weekly_review").
		Description("Comprehensive weekly review to assess productivity, adjust priorities, and plan the upcoming week.").
		Handler(func(ctx context.Context, args map[string]string) (*mcp.PromptResult, error) {
			return &mcp.PromptResult{
				Description: "Weekly Review Session",
				Messages: []mcp.PromptMessage{
					{
						Role: string(mcp.RoleUser),
						Content: mcp.TextContent{
							Type: "text",
							Text: `Let's conduct a weekly review. Please:

1. Check the productivity summary using orbita://insights/summary
2. Review all tasks using orbita://tasks resource
3. Check this week's schedule using orbita://schedule/week
4. Review goals progress using orbita://insights/goals

Help me analyze:

**Accomplishments:**
- What tasks were completed this week?
- What habits were maintained consistently?
- What goals made progress?

**Challenges:**
- What tasks are overdue and why?
- What habits were missed?
- Were there any scheduling conflicts?

**Planning:**
- What should be the top priorities for next week?
- Are there any tasks that should be broken down?
- Should any recurring tasks be adjusted?

**Optimization:**
- Based on the data, what time of day am I most productive?
- Are there patterns in missed habits or overdue tasks?
- What adjustments would improve next week?

Please provide specific recommendations with actionable next steps.`,
						},
					},
				},
			}, nil
		})

	// Task breakdown prompt
	srv.Prompt("task_breakdown").
		Description("Break down a complex task into smaller, manageable subtasks with time estimates.").
		Argument("task_description", "Description of the task to break down", true).
		Handler(func(ctx context.Context, args map[string]string) (*mcp.PromptResult, error) {
			taskDesc := args["task_description"]
			if taskDesc == "" {
				taskDesc = "[Please describe the task you want to break down]"
			}

			return &mcp.PromptResult{
				Description: "Task Breakdown Assistant",
				Messages: []mcp.PromptMessage{
					{
						Role: string(mcp.RoleUser),
						Content: mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf(`Help me break down this task into smaller, actionable subtasks:

**Task:** %s

Please:
1. Analyze the task and identify the main components
2. Break it into 3-7 subtasks that can each be completed in one sitting
3. For each subtask, suggest:
   - A clear, action-oriented title
   - Estimated duration in minutes (15, 30, 45, 60, 90, or 120)
   - Priority (high, medium, low)
   - Any dependencies on other subtasks

4. Suggest an optimal order to complete the subtasks
5. Identify any potential blockers or resources needed

Once I approve the breakdown, use the task.create tool to create each subtask.
Use a consistent naming pattern like "[Parent Task] - Subtask Name".`, taskDesc),
						},
					},
				},
			}, nil
		})

	// Focus session prompt
	srv.Prompt("focus_session").
		Description("Start a focused work session with a specific task, minimizing distractions.").
		Argument("duration", "Session duration in minutes (default: 25 for Pomodoro)", false).
		Handler(func(ctx context.Context, args map[string]string) (*mcp.PromptResult, error) {
			duration := args["duration"]
			if duration == "" {
				duration = "25"
			}

			return &mcp.PromptResult{
				Description: "Focus Session Setup",
				Messages: []mcp.PromptMessage{
					{
						Role: string(mcp.RoleUser),
						Content: mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf(`I want to start a %s-minute focus session. Please help me:

1. Review my active tasks using orbita://tasks/active
2. Check today's schedule using orbita://schedule/today
3. Identify the highest priority task that fits in %s minutes

Then:
- Confirm which task I'll focus on
- Clear any scheduling conflicts for the next %s minutes
- Set up the focus session

During the session:
- I should avoid checking other tasks
- Any new thoughts/tasks should be captured to inbox
- Breaks should follow the Pomodoro technique if applicable

After confirmation, use the appropriate tools to:
1. Block time on my schedule for the focus session
2. Optionally start a focus mode timer if available

Let me know what task you recommend for this session and why.`, duration, duration, duration),
						},
					},
				},
			}, nil
		})

	// Inbox processing prompt
	srv.Prompt("inbox_zero").
		Description("Process inbox items efficiently using the GTD methodology.").
		Handler(func(ctx context.Context, args map[string]string) (*mcp.PromptResult, error) {
			return &mcp.PromptResult{
				Description: "Inbox Zero Processing",
				Messages: []mcp.PromptMessage{
					{
						Role: string(mcp.RoleUser),
						Content: mcp.TextContent{
							Type: "text",
							Text: `Let's process my inbox to zero. For each item, help me decide:

1. First, list all inbox items using inbox.list

2. For each item, guide me through these questions:
   - Is it actionable?
     - NO → Delete, Archive, or add to Reference
     - YES → Continue...

   - Can it be done in under 2 minutes?
     - YES → Do it now
     - NO → Continue...

   - Am I the right person to do this?
     - NO → Delegate it
     - YES → Continue...

   - Does it have a deadline?
     - YES → Create task with due date
     - NO → Add to task list or Someday/Maybe

3. For items that become tasks:
   - Help me set appropriate priority (high/medium/low)
   - Estimate duration
   - Suggest scheduling if urgent

4. Track progress: Show me how many items we've processed and how many remain

Use the inbox.process, task.create, and other tools as needed.
Let's start with the first inbox item.`,
						},
					},
				},
			}, nil
		})

	// Habit setup prompt
	srv.Prompt("habit_setup").
		Description("Create a new habit with optimal scheduling and tracking strategy.").
		Argument("habit_name", "Name of the habit you want to build", true).
		Argument("frequency", "How often (daily, weekly, specific days)", false).
		Handler(func(ctx context.Context, args map[string]string) (*mcp.PromptResult, error) {
			habitName := args["habit_name"]
			if habitName == "" {
				habitName = "[Please specify the habit you want to build]"
			}
			frequency := args["frequency"]
			if frequency == "" {
				frequency = "daily"
			}

			return &mcp.PromptResult{
				Description: "Habit Setup Assistant",
				Messages: []mcp.PromptMessage{
					{
						Role: string(mcp.RoleUser),
						Content: mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf(`Help me set up a new habit effectively:

**Habit:** %s
**Target Frequency:** %s

1. First, review my existing habits using orbita://habits/active
2. Check my typical schedule using orbita://schedule/week

Based on this, help me:

**Design the Habit:**
- Suggest an optimal time of day based on my schedule
- Recommend an appropriate duration (start small!)
- Identify potential habit stacking opportunities (link to existing habits)

**Set Success Criteria:**
- Define what "done" looks like for this habit
- Set a realistic streak goal to start
- Plan for the "two-day rule" (never miss twice)

**Handle Obstacles:**
- What might prevent me from doing this habit?
- Create if-then plans for common obstacles
- Identify an accountability mechanism

**Create the Habit:**
Once I confirm the details, use habit.create with:
- Clear name and description
- Optimal schedule/time
- Appropriate duration
- Any linked habits or tasks

Also suggest initial reminders and tracking approach.`, habitName, frequency),
						},
					},
				},
			}, nil
		})

	// Meeting preparation prompt
	srv.Prompt("meeting_prep").
		Description("Prepare for an upcoming meeting with agenda, context, and action items.").
		Argument("meeting_topic", "Topic or title of the meeting", true).
		Argument("attendees", "List of attendees (optional)", false).
		Handler(func(ctx context.Context, args map[string]string) (*mcp.PromptResult, error) {
			topic := args["meeting_topic"]
			if topic == "" {
				topic = "[Please specify the meeting topic]"
			}
			attendees := args["attendees"]
			if attendees == "" {
				attendees = "Not specified"
			}

			return &mcp.PromptResult{
				Description: "Meeting Preparation Assistant",
				Messages: []mcp.PromptMessage{
					{
						Role: string(mcp.RoleUser),
						Content: mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf(`Help me prepare for this meeting:

**Meeting Topic:** %s
**Attendees:** %s

Please:

1. Review related tasks using task.list (search for relevant keywords)
2. Check my recent schedule for context

**Preparation Checklist:**
- [ ] Define clear meeting objectives (what decisions/outcomes needed?)
- [ ] Create an agenda with time allocations
- [ ] Identify relevant background information to share
- [ ] List questions I need answers to
- [ ] Prepare any materials/documents needed

**Action Items Framework:**
For any action items that come up, I'll need:
- Clear owner
- Specific deliverable
- Due date

**Post-Meeting Tasks:**
Help me create tasks for:
- Sending meeting notes
- Following up on action items
- Scheduling any follow-up meetings

Once we've prepared, create any necessary tasks using task.create
and optionally schedule a prep time block before the meeting.`, topic, attendees),
						},
					},
				},
			}, nil
		})

	// Energy management prompt
	srv.Prompt("energy_check").
		Description("Log your current energy level and get task recommendations that match.").
		Argument("energy_level", "Current energy: high, medium, or low", true).
		Handler(func(ctx context.Context, args map[string]string) (*mcp.PromptResult, error) {
			energy := args["energy_level"]
			if energy == "" {
				energy = "medium"
			}

			return &mcp.PromptResult{
				Description: "Energy-Based Task Matching",
				Messages: []mcp.PromptMessage{
					{
						Role: string(mcp.RoleUser),
						Content: mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf(`My current energy level is: **%s**

Please:
1. Review my active tasks using orbita://tasks/active
2. Check today's remaining schedule using orbita://schedule/today

Based on my energy level, recommend tasks that match:

**High Energy:** Complex tasks requiring deep focus, creative work, difficult conversations
**Medium Energy:** Administrative tasks, routine work, meetings, planning
**Low Energy:** Simple tasks, organizing, reading, light communication

For my current %s energy:
- List 3-5 tasks I should tackle now
- Explain why each matches my energy level
- Suggest estimated time for each

Also consider:
- Task deadlines (urgent items may need attention regardless of energy)
- Time of day and typical energy patterns
- Any scheduled commitments coming up

If my energy is low but I have high-priority tasks:
- Suggest energy boosters (short walk, snack, power nap)
- Recommend breaking large tasks into smaller chunks
- Identify if any tasks can be deferred

Would you like me to log this energy level using wellness tools if available?`, energy, energy),
						},
					},
				},
			}, nil
		})

	// Quick capture prompt
	srv.Prompt("quick_capture").
		Description("Quickly capture a thought, idea, or task to process later.").
		Argument("content", "What you want to capture", true).
		Handler(func(ctx context.Context, args map[string]string) (*mcp.PromptResult, error) {
			content := args["content"]
			if content == "" {
				content = "[Please specify what you want to capture]"
			}

			return &mcp.PromptResult{
				Description: "Quick Capture",
				Messages: []mcp.PromptMessage{
					{
						Role: string(mcp.RoleUser),
						Content: mcp.TextContent{
							Type: "text",
							Text: fmt.Sprintf(`Quick capture this item: "%s"

Please analyze and help me:

1. **Categorize** - Is this:
   - A task (something to do)?
   - An idea (something to explore)?
   - A reference (something to remember)?
   - A calendar item (something scheduled)?

2. **Process** appropriately:
   - **Task**: Use task.create with suggested priority and duration
   - **Idea**: Add to inbox with "idea:" prefix for later processing
   - **Reference**: Add to inbox with "ref:" prefix
   - **Calendar**: Suggest using schedule tools

3. **Enrich** if possible:
   - Add relevant context
   - Suggest related existing tasks
   - Estimate priority based on keywords

4. **Confirm** the action taken

This should be fast - capture now, process properly during inbox review.
Use inbox.add for anything that needs more thought later.`, content),
						},
					},
				},
			}, nil
		})

	return nil
}
