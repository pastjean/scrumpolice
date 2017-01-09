package scrumpolice

type Report struct {
	User                string
	Current             int
	Complete            bool
	Questions           []string
	QuestionsAndAnswers map[string]string
}

func NewReport(user string, questions []string) *Report {
	return &Report{user, 0, false, questions, map[string]string{}}
}

func (report *Report) ResetQuestions() string {
	report.Current = 0
	report.Complete = false
	return report.Questions[report.Current]
}

func (report *Report) CurrentQuestion() string {
	if report.Complete == true {
		return ""
	}
	return report.Questions[report.Current]
}

func (report *Report) NextQuestion() string {
	if report.Current == len(report.Questions)-1 {
		report.Complete = true
		return ""
	}
	report.Current += 1
	return report.Questions[report.Current]
}
