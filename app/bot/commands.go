package bot

type Command int

const (
	Help Command = iota
	Bind
	Register
	Deregister
	List
	Score
	AC
	UndoAC
	AddProblem
	RemoveProblem
	ClearClass
	CreateClass
	Users
	Classes
	UpdateProblem
	MyClasses
	ClassesOf
	Me
	ChangeUsername
	Exterminate
	Export
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
		Cmd:        "!help",
		Name:        "help",
		Usage:       "!help [command]",
		Description: "Show info for all commands or details for a specific command",
		UserRoles:   NewUserRoleSet(RoleAll),
	},
	Bind: {
		Cmd:        "!bind",
		Name:        "bind",
		Usage:       "!bind <user_name>",
		Description: "Bind your Discord account to a username for tracking progress",
		UserRoles:   NewUserRoleSet(RoleAll),
		ExtraInfo:   "The `user_name` parameter should be one word (e.g., \"jorge_ibanez\" or \"jorgeibanez\")",
	},
	ChangeUsername: {
		Cmd:        "!change_username",
		Name:        "change_username",
		Usage:       "!change_username <user_name>",
		Description: "Change your current username to a new one",
		UserRoles:   NewUserRoleSet(RoleAll),
		ExtraInfo:   "The `user_name` parameter should be one word (e.g., \"jorge_ibanez\" or \"jorgeibanez\"). If you don't have a username bound yet, this will bind the new username to your account.",
	},
	Register: {
		Cmd:        "!register",
		Name:        "register",
		Usage:       "!register <class_code>\n!register <user_name> <class_code>",
		Description: "Register yourself or another user in a class",
		UserRoles:   NewUserRoleSet(RoleAll, RoleMentor),
		ExtraInfo: "**As Regular User:** Use `!register <class_code>` to register yourself in a class.\n" +
			"**As Mentor:** Use `!register <user_name> <class_code>` to register another user in a class.\n" +
			"The `class_code` and `user_name` parameters should be one word (e.g., \"basicos_1\", \"jorge_ibanez\")",
	},
	Deregister: {
		Cmd:        "!deregister",
		Name:        "deregister",
		Usage:       "!deregister <class_code>\n!deregister <user_name> <class_code>",
		Description: "Remove yourself or another user from a class",
		UserRoles:   NewUserRoleSet(RoleEnrolled, RoleMentor),
		ExtraInfo: "**As Enrolled User:** Use `!deregister <class_code>` to remove yourself from a class.\n" +
			"**As Mentor:** Use `!deregister <user_name> <class_code>` to remove another user from a class.\n" +
			"⚠️ **WARNING:** This will delete ALL solved problems and pending confirmations for the user in this class.",
	},
	List: {
		Cmd:        "!list",
		Name:        "list",
		Usage:       "!list <class_code>",
		Description: "Show the list of problems available for the specified class",
		UserRoles:   NewUserRoleSet(RoleAll),
		ExtraInfo:   "The `class_code` parameter should be one word (e.g., \"basicos_1\")",
	},
	Score: {
		Cmd:        "!score",
		Name:        "score",
		Usage:       "!score [class_code] [user_name]",
		Description: "Show scores for all classes, a specific class, or a specific user in a class",
		UserRoles:   NewUserRoleSet(RoleAll),
		ExtraInfo:   "Both `class_code` and `user_name` parameters should be one word. You can use `!score` to see all scores, `!score class_code` to see scores for a class, or `!score user_name class_code` to see a specific user's scores in a class.",
	},
	AC: {
		Cmd:        "!ac",
		Name:        "ac",
		Usage:       "!ac <problem_id>\n!ac <user_name> <problem_id>",
		Description: "Mark a problem as solved for yourself or another user",
		UserRoles:   NewUserRoleSet(RoleEnrolled, RoleMentor),
		ExtraInfo: "**As Enrolled User:** Use `!ac <problem_id>` to mark a problem as solved (requires mentor approval with ✅ or ❌).\n" +
			"**As Mentor:** Use `!ac <user_name> <problem_id>` to directly mark a problem as solved for another user (no approval needed).",
	},
	UndoAC: {
		Cmd:        "!undo_ac",
		Name:        "undo_ac",
		Usage:       "!undo_ac <problem_id>\n!undo_ac <user_name> <problem_id>",
		Description: "Remove a previously solved problem from your or another user's solved list",
		UserRoles:   NewUserRoleSet(RoleEnrolled, RoleMentor),
		ExtraInfo: "**As Enrolled User:** Use `!undo_ac <problem_id>` to remove a problem from your own solved list.\n" +
			"**As Mentor:** Use `!undo_ac <user_name> <problem_id>` to remove a problem from another user's solved list.",
	},
	AddProblem: {
		Cmd:        "!add_problem",
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
		Cmd:        "!remove_problem",
		Name:        "remove_problem",
		Usage:       "!remove_problem <problem_id>",
		Description: "Remove a problem from a class by its ID",
		UserRoles:   NewUserRoleSet(RoleMentor),
	},
	UpdateProblem: {
		Cmd:        "!update_problem",
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
		Cmd:        "!clear_class",
		Name:        "clear_class",
		Usage:       "!clear_class <class_code>",
		Description: "Remove all problems, enrollments, and scores from a class",
		UserRoles:   NewUserRoleSet(RoleAdmin),
		ExtraInfo:   "The `class_code` parameter should be one word (e.g., \"basicos_1\")",
	},
	CreateClass: {
		Cmd:        "!create_class",
		Name:        "create_class",
		Usage:       "!create_class <class_code> <ClassName>",
		Description: "Create a new class with the specified code and name",
		UserRoles:   NewUserRoleSet(RoleAdmin),
		ExtraInfo:   "The `class_code` should be one word, while `ClassName` can include multiple words (e.g., \"basicos_1 Basicos 2025\")",
	},
	Exterminate: {
		Cmd:        "!exterminate",
		Name:        "exterminate",
		Usage:       "!exterminate <user_id>",
		Description: "Remove user from all system data including bindings, enrollments, solved problems and pending confirmations",
		UserRoles:   NewUserRoleSet(RoleAdmin),
		ExtraInfo:   "The `user_id` should be the Discord user ID (e.g., \"123456789012345678\"). This action cannot be undone.",
	},
	Export: {
		Cmd:        "!export",
		Name:        "export",
		Usage:       "!export <class_code>",
		Description: "Export all class data (users, problems, scores) to an Excel file",
		UserRoles:   NewUserRoleSet(RoleAdmin),
		ExtraInfo:   "The `class_code` parameter should be one word (e.g., \"basicos_1\"). The Excel file will contain three sheets: Users, Problems, and Scores.",
	},
	Users: {
		Cmd:        "!users",
		Name:        "users",
		Usage:       "!users <class_code>",
		Description: "Show a list of all users registered in the specified class",
		UserRoles:   NewUserRoleSet(RoleAll),
		ExtraInfo:   "The `class_code` parameter should be one word (e.g., \"basicos_1\")",
	},
	Classes: {
		Cmd:        "!classes",
		Name:        "classes",
		Usage:       "!classes",
		Description: "Show a list of all available classes",
		UserRoles:   NewUserRoleSet(RoleAll),
	},
	MyClasses: {
		Cmd:        "!my_classes",
		Name:        "my_classes",
		Usage:       "!my_classes",
		Description: "Show a list of all classes you're currently enrolled in",
		UserRoles:   NewUserRoleSet(RoleEnrolled),
	},
	ClassesOf: {
		Cmd:        "!classes_of",
		Name:        "classes_of",
		Usage:       "!classes_of <user_name>",
		Description: "Show a list of all classes a specific user is enrolled in",
		UserRoles:   NewUserRoleSet(RoleAll),
		ExtraInfo:   "The `user_name` parameter should be one word (e.g., \"jorge_ibanez\" or \"jorgeibanez\")",
	},
	Me: {
		Cmd:        "!me",
		Name:        "me",
		Usage:       "!me",
		Description: "Show the username bound to your Discord account",
		UserRoles:   NewUserRoleSet(RoleAll),
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
