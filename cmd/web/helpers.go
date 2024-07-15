package main

func (app *Config) sendemail(msg Message) {
	app.Wait.Add(1)
	app.Mailer.MailertChan <- msg
}
