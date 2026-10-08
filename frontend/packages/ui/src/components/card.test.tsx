import { render, screen } from "@testing-library/react";
import { Card, CardContent, CardHeader, CardTitle } from "#components/card";

test("renders title and content", () => {
  render(
    <Card>
      <CardHeader>
        <CardTitle>Title</CardTitle>
      </CardHeader>
      <CardContent>Body</CardContent>
    </Card>,
  );
  expect(screen.getByText("Title")).toBeInTheDocument();
  expect(screen.getByText("Body")).toBeInTheDocument();
});
