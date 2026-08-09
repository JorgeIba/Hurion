package bot

type Command int

const (
	Help Command = iota
	List
	Scores
	Score
	ScoreOf
	AC
	UndoAC
	AddProblem
	RemoveProblem
	ClearClass
	CreateClass
	LinkRoles
	LinkRooms
	GetRoles
	GetChannels
	Users
	Classes
	UpdateProblem
	MyClasses
	ClassesOf
	Exterminate
	ExterminateClass
	Export
	ScoreImage
)

type CommandData struct {
	Cmd         string
	Name        string
	Usage       string
	Description string
	UserRoles   UserRoleSet
	ExtraInfo   string
}

var Commands = map[Command]CommandData{
	Help: {
		Cmd:         "!help",
		Name:        "help",
		Usage:       "!help [command]",
		Description: "Show info for all commands or details for a specific command",
		UserRoles:   NewUserRoleSet(RoleAll),
	},
	List: {
		Cmd:         "!list",
		Name:        "list",
		Usage:       "!list [class_code]",
		Description: "Show the list of problems available for a class or infer class from the current channel",
		UserRoles:   NewUserRoleSet(RoleAll),
		ExtraInfo:   "The optional `class_code` parameter should be one word (e.g., \"basicos_1\"). If omitted, the bot uses the current channel context.",
	},
	Score: {
		Cmd:         "!score",
		Name:        "score",
		Usage:       "!score [class_code]",
		Description: "Get link to real-time interactive scoreboard Web UI for a class",
		UserRoles:   NewUserRoleSet(RoleAll),
		ExtraInfo:   "The optional `class_code` parameter should be one word (e.g., \"basicos_1\"). If omitted, the bot uses the current channel context.",
	},
	ScoreImage: {
		Cmd:         "!score_image",
		Name:        "score_image",
		Usage:       "!score_image [class_code]",
		Description: "Generate and attach static PNG image table of scores for a class",
		UserRoles:   NewUserRoleSet(RoleAll),
		ExtraInfo:   "Renders a static PNG table of class scores. Note: For large classes, use `!score` to view the interactive Web UI.",
	},
	Scores: {
		Cmd:         "!scores",
		Name:        "scores",
		Usage:       "!scores",
		Description: "Show scores for all classes as image tables",
		UserRoles:   NewUserRoleSet(RoleAll),
	},
	ScoreOf: {
		Cmd:         "!score_of",
		Name:        "score_of",
		Usage:       "!score_of <@user> [class_code]",
		Description: "Show a user's score in a class or infer class from the current channel",
		UserRoles:   NewUserRoleSet(RoleAll),
		ExtraInfo:   "Mention the user with @ or provide a raw Discord ID. If `class_code` is omitted, the bot uses the current channel context.",
	},
	AC: {
		Cmd:         "!ac",
		Name:        "ac",
		Usage:       "!ac <problem_id>\n!ac <@user> <problem_id>",
		Description: "Mark a problem as solved for yourself or another user",
		UserRoles:   NewUserRoleSet(RoleEnrolled, RoleMentor),
		ExtraInfo: "**As Enrolled User:** Use `!ac <problem_id>` to mark a problem as solved (requires mentor approval with ✅ or ❌).\n" +
			"**As Mentor:** Use `!ac <@user> <problem_id>` to directly mark a problem as solved for another user (no approval needed).",
	},
	UndoAC: {
		Cmd:         "!undo_ac",
		Name:        "undo_ac",
		Usage:       "!undo_ac <problem_id>\n!undo_ac <@user> <problem_id>",
		Description: "Remove a previously solved problem from your or another user's solved list",
		UserRoles:   NewUserRoleSet(RoleEnrolled, RoleMentor),
		ExtraInfo: "**As Enrolled User:** Use `!undo_ac <problem_id>` to remove a problem from your own solved list.\n" +
			"**As Mentor:** Use `!undo_ac <@user> <problem_id>` to remove a problem from another user's solved list.",
	},
	AddProblem: {
		Cmd:         "!add_problem",
		Name:        "add_problem",
		Usage:       "!add_problem <class_code> <url> [difficulty] [topic] <ProblemName>",
		Description: "Add a new problem to a class with URL, optional difficulty and topic, and name",
		UserRoles:   NewUserRoleSet(RoleMentor),
		ExtraInfo: "The `class_code` should be one word, url should have 'http://' prefix (Discord policies I guess), " +
			"`difficulty` can be: Easy, Medium, Hard, or use '-' to skip (No case sensitive). " +
			"`topic` can be any text, one word, use '-' to skip (No case sensitive). " +
			"`ProblemName` can include multiple words.\n" +
			"Examples:\n" +
			"  !add_problem basicos http://example.com - - Two Sum (no difficulty, no topic)\n" +
			"  !add_problem basicos http://example.com Easy - Two Sum (only difficulty)\n" +
			"  !add_problem basicos http://example.com - DP Two Sum (only topic)\n" +
			"  !add_problem basicos http://example.com Medium Graphs Binary Tree Problem (both)",
	},
	RemoveProblem: {
		Cmd:         "!remove_problem",
		Name:        "remove_problem",
		Usage:       "!remove_problem <problem_id>",
		Description: "Remove a problem from a class by its ID",
		UserRoles:   NewUserRoleSet(RoleMentor),
	},
	UpdateProblem: {
		Cmd:         "!update_problem",
		Name:        "update_problem",
		Usage:       "!update_problem <problem_id> <class_code> <url> [difficulty] [topic] <ProblemName>",
		Description: "Update a problem's details while preserving its ID for AC records",
		UserRoles:   NewUserRoleSet(RoleMentor),
		ExtraInfo: "The `problem_id` and `class_code` should be one word, url should have 'http://' prefix (Discord policies I guess), " +
			"`difficulty` can be: Easy, Medium, Hard, or use '-' to skip (No case sensitive). " +
			"`topic` can be any text, one word, use '-' to skip (No case sensitive). " +
			"`ProblemName` can include multiple words.\n" +
			"Examples:\n" +
			"  !update_problem 3 basicos http://example.com - - Updated Name (no difficulty, no topic)\n" +
			"  !update_problem 3 basicos http://example.com Hard - Updated Name (only difficulty)\n" +
			"  !update_problem 3 basicos http://example.com - DP Updated Name (only topic)\n" +
			"  !update_problem 3 basicos http://example.com Easy Graphs Updated Problem Name (both)",
	},
	ClearClass: {
		Cmd:         "!clear_class",
		Name:        "clear_class",
		Usage:       "!clear_class [class_code]",
		Description: "Reset a class by removing problems, solved records, and score history while keeping the class, rooms, and roles",
		UserRoles:   NewUserRoleSet(RoleAdmin),
		ExtraInfo:   "The optional `class_code` parameter should be one word (e.g., \"basicos_1\"). If omitted, the bot uses the current channel context.",
	},
	CreateClass: {
		Cmd:         "!create_class",
		Name:        "create_class",
		Usage:       "!create_class <class_code> <ClassName>",
		Description: "Create a new class with the specified code and name",
		UserRoles:   NewUserRoleSet(RoleAdmin),
		ExtraInfo:   "The `class_code` should be one word, while `ClassName` can include multiple words (e.g., \"basicos_1 Basicos 2025\")",
	},
	LinkRoles: {
		Cmd:         "!link_roles",
		Name:        "link_roles",
		Usage:       "!link_roles <class_code> <@role1> [@role2] ...",
		Description: "Link Discord roles to a class for access control",
		UserRoles:   NewUserRoleSet(RoleAdmin),
		ExtraInfo:   "Allows users with these roles to submit ACs for the class. Mention multiple roles separated by spaces.",
	},
	LinkRooms: {
		Cmd:         "!link_rooms",
		Name:        "link_rooms",
		Usage:       "!link_rooms <class_code> <#channel1> [#channel2] ...",
		Description: "Link Discord channels to a class as classrooms",
		UserRoles:   NewUserRoleSet(RoleAdmin),
		ExtraInfo:   "Sets up default class context for commands run in these channels. Mention multiple channels separated by spaces.",
	},
	GetRoles: {
		Cmd:         "!get_roles",
		Name:        "get_roles",
		Usage:       "!get_roles [class_code]",
		Description: "Show all Discord roles linked to a class or infer class from channel",
		UserRoles:   NewUserRoleSet(RoleAll),
	},
	GetChannels: {
		Cmd:         "!get_channels",
		Name:        "get_channels",
		Usage:       "!get_channels [class_code]",
		Description: "Show all Discord channels linked to a class or infer class from channel",
		UserRoles:   NewUserRoleSet(RoleAll),
	},
	Exterminate: {
		Cmd:         "!exterminate",
		Name:        "exterminate",
		Usage:       "!exterminate <@user>",
		Description: "Remove user from all system data including solved problems and pending confirmations",
		UserRoles:   NewUserRoleSet(RoleAdmin),
		ExtraInfo:   "Mention the user with @ (e.g., \"@DaddyYorch\"). This action cannot be undone.",
	},
	ExterminateClass: {
		Cmd:         "!exterminate_class",
		Name:        "exterminate_class",
		Usage:       "!exterminate_class <class_code>",
		Description: "Permanently delete a class and all of its linked data and settings",
		UserRoles:   NewUserRoleSet(RoleAdmin),
		ExtraInfo:   "The `class_code` parameter should be one word (e.g., \"basicos_1\"). This action cannot be undone.",
	},
	Export: {
		Cmd:         "!export",
		Name:        "export",
		Usage:       "!export <class_code>",
		Description: "Export all class data (users, problems, scores) to an Excel file",
		UserRoles:   NewUserRoleSet(RoleAdmin),
		ExtraInfo:   "The `class_code` parameter should be one word (e.g., \"basicos_1\"). The Excel file will contain three sheets: Users, Problems, and Scores.",
	},
	Users: {
		Cmd:         "!users",
		Name:        "users",
		Usage:       "!users <class_code>",
		Description: "Show a list of all users registered in the specified class",
		UserRoles:   NewUserRoleSet(RoleAll),
		ExtraInfo:   "The `class_code` parameter should be one word (e.g., \"basicos_1\")",
	},
	Classes: {
		Cmd:         "!classes",
		Name:        "classes",
		Usage:       "!classes",
		Description: "Show a list of all available classes",
		UserRoles:   NewUserRoleSet(RoleAll),
	},
	MyClasses: {
		Cmd:         "!my_classes",
		Name:        "my_classes",
		Usage:       "!my_classes",
		Description: "Show a list of all classes you're currently enrolled in",
		UserRoles:   NewUserRoleSet(RoleEnrolled),
	},
	ClassesOf: {
		Cmd:         "!classes_of",
		Name:        "classes_of",
		Usage:       "!classes_of <@user>",
		Description: "Show a list of all classes a specific user is enrolled in",
		UserRoles:   NewUserRoleSet(RoleAll),
		ExtraInfo:   "Mention the user with @ (e.g., \"@DaddyYorch\").",
	},
}

var (
	commandsByFull map[string]Command
	commandsByName map[string]Command
)

func init() {
	commandsByFull = make(map[string]Command, len(Commands))
	commandsByName = make(map[string]Command, len(Commands))
	for cmd, data := range Commands {
		commandsByFull[data.Cmd] = cmd
		commandsByName[data.Name] = cmd
	}
}

func (c Command) String() string {
	return Commands[c].Cmd
}

func (c Command) Name() string {
	return Commands[c].Name
}

func (c Command) Data() CommandData {
	return Commands[c]
}

func CommandByFull(full string) (Command, bool) {
	cmd, ok := commandsByFull[full]
	return cmd, ok
}

func CommandByName(name string) (Command, bool) {
	cmd, ok := commandsByName[name]
	return cmd, ok
}
