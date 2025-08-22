/*
Open a file, directory, or URI using the OS's default
application for that object type. You can optionally specify
an application to use.

This is a proxy for the following commands:

Darwin: "open"
Linux: "xdg-open"
Windows: "start"
*/
package xdg

/*
Open a file, directory, or URI using the OS's default
application for that object type. Wait for the open
command to complete.
*/
func Run(input string) error {
	return open(input).Run()
}

/*
Open a file, directory, or URI using the OS's default
application for that object type. Don't wait for the
open command to complete.
*/
func Start(input string) error {
	c := open(input)
	go c.Wait()
	return c.Start()
}

/*
Open a file, directory, or URI using the specified application.
Wait for the open command to complete.
*/
func RunWith(input string, appName string) error {
	return openWith(input, appName).Run()
}

/*
Open a file, directory, or URI using the specified application.
Don't wait for the open command to complete.
*/
func StartWith(input string, appName string) error {
	c := openWith(input, appName)
	go c.Wait()
	return c.Start()
}
